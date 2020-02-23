package iavl

import (
	"bytes"

	"github.com/pkg/errors"

	db "github.com/tendermint/tm-db"
)

var ErrNoImport = errors.New("no import in progress")

// Importer imports data into an empty MutableTree. It is created by MutableTree.Import(). Users
// must call Close() when done.
//
// Importer is not concurrency-safe, it is the caller's responsibility to ensure the tree is not
// modified while performing an import.
type Importer struct {
	tree    *MutableTree
	version int64
	batch   db.Batch
	stack   []*Node
}

// newImporter creates a new Importer for an empty MutableTree.
//
// version should correspond to the version that was initially exported. It must be greater than
// or equal to the highest ExportNode version number given.
func newImporter(tree *MutableTree, version int64) (*Importer, error) {
	if version < 0 {
		return nil, errors.New("imported version cannot be negative")
	}
	if tree.ndb.latestVersion > 0 {
		return nil, errors.Errorf("found database at version %d, must be 0", tree.ndb.latestVersion)
	}
	if !tree.IsEmpty() {
		return nil, errors.New("tree must be empty")
	}

	return &Importer{
		tree:    tree,
		version: version,
		batch:   tree.ndb.snapshotDB.NewBatch(),
		stack:   make([]*Node, 0, 8),
	}, nil
}

// Close frees all resources, discarding any uncommitted nodes. It is safe to call multiple times.
func (i *Importer) Close() {
	i.batch.Close()
	i.tree = nil
}

// Add adds an ExportNode to the import. ExportNodes must be added in the order returned by
// Exporter, i.e. depth-first post-order (LRN).
func (i *Importer) Add(exportNode *ExportNode) error {
	if i.tree == nil {
		return ErrNoImport
	}
	if exportNode == nil {
		return errors.New("node cannot be nil")
	}
	if exportNode.Version > i.version {
		return errors.Errorf("node version %v can't be greater than import version %v",
			exportNode.Version, i.version)
	}

	node := &Node{
		key:     exportNode.Key,
		value:   exportNode.Value,
		version: exportNode.Version,
		height:  exportNode.Height,
	}

	// We build the tree from the bottom-left up. The stack is used to store unresolved left
	// children while constructing right children. When all children are built, the parent can
	// be constructed and the resolved children can be discarded from the stack. Using a stack
	// ensures that we can handle additional unresolved left children while building a right branch.
	//
	// We don't modify the stack until we've verified the built node, to avoid leaving the
	// importer in an inconsistent state when we return an error.
	stackSize := len(i.stack)
	switch {
	case stackSize >= 2 && i.stack[stackSize-1].height < node.height && i.stack[stackSize-2].height < node.height:
		node.leftNode = i.stack[stackSize-2]
		node.leftHash = node.leftNode.hash
		node.rightNode = i.stack[stackSize-1]
		node.rightHash = node.rightNode.hash
	case stackSize >= 1 && i.stack[stackSize-1].height < node.height:
		node.leftNode = i.stack[stackSize-1]
		node.leftHash = node.leftNode.hash
	}

	if node.height == 0 {
		node.size = 1
	}
	if node.leftNode != nil {
		node.size += node.leftNode.size
	}
	if node.rightNode != nil {
		node.size += node.rightNode.size
	}

	node._hash()
	err := node.validate()
	if err != nil {
		return err
	}

	var buf bytes.Buffer
	err = node.writeBytes(&buf)
	if err != nil {
		return err
	}

	// FIXME Can we build one giant batch per import, or do we have to flush it regularly?
	i.batch.Set(i.tree.ndb.nodeKey(node.hash), buf.Bytes())

	// Update the stack now that we know there were no errors
	switch {
	case node.leftHash != nil && node.rightHash != nil:
		i.stack = i.stack[:stackSize-2]
	case node.leftHash != nil || node.rightHash != nil:
		i.stack = i.stack[:stackSize-1]
	}
	i.stack = append(i.stack, node)

	return nil
}

// Commit commits the import, writing it to the database. It can only be called once, and calls
// Close() internally.
func (i *Importer) Commit() error {
	if i.tree == nil {
		return ErrNoImport
	}

	switch {
	case len(i.stack) == 1:
		i.batch.Set(i.tree.ndb.rootKey(i.version), i.stack[0].hash)
	case len(i.stack) > 2:
		return errors.Errorf("invalid node structure, found stack size %v when finalizing",
			len(i.stack))
	}

	err := i.batch.WriteSync()
	if err != nil {
		return err
	}
	i.tree.ndb.updateLatestVersion(i.version)

	root, err := i.tree.ndb.getRoot(i.version)
	if err != nil {
		return err
	}
	if len(root) > 0 {
		i.tree.ImmutableTree.root = i.tree.ndb.GetNode(root)
	}

	i.tree.versions[i.version] = true
	i.tree.version = i.version

	if len(root) > 0 {
		last, err := i.tree.GetImmutable(i.version)
		if err != nil {
			return err
		}
		i.tree.lastSaved = last
	}

	i.Close()
	return nil
}
