What does a merkle proof include?
How does a proof prove that a value is present in the tree?
What kinds of proofs are there (for leaf nodes, for inner nodes, root hash, etc)?
Components that still need writeups:

- RangeProof
- Components of Proofs (e.g. ProofInnerNode, PathToLeaf)
- Proof of Value
- Proof of Abscence
- Proof Operator abstraction


# Proof

Part of the purpose of a IAVL tree is to provide the ability to return proofs along values, which can
be later used to verify if the provided proof is indeed valid from the IAVL merkle tree.

Users can either call GetWithProof/GetVersionedWithProof or GetRangeWithProof/GetRangeVersionedWithProof 
api/method call that provides a key in the request, and IAVL will return the value along with a RangeProof.
Later section will explain in more details what consists of a Range proof, and how IAVL produces and verifies it.
With a range proof, IAVL tree provides APIs to verify if either 1) the range proof is valid (Verify) 2) an item key/value exists (VerifyItem)
3) an item key/value doesn't exist (VerifyAbsence).

## RangeProof

As specified in the earlier section, a RangeProof is the object that is returned by the IAVL when requesting a key with proof.
A RangeProof is defined as the following:

```golang
type RangeProof struct {
	LeftPath   PathToLeaf      `json:"left_path"`
	InnerNodes []PathToLeaf    `json:"inner_nodes"`
	Leaves     []ProofLeafNode `json:"leaves"`
}
```

To fully understand this, we will first understand what PathToLeaf and ProofInnerNode is.

### ProofInnerNode

ProofInnerNode is a struct that simply holds information about a node in the IAVL tree.
It records the following information:

```golang
type ProofInnerNode struct {
	Height  int8   `json:"height"`
	Size    int64  `json:"size"`
	Version int64  `json:"version"`
	Left    []byte `json:"left"`
	Right   []byte `json:"right"`
}
```

This holds a partial information of what is already storing in `Node` struct. For information about what each field mean
please refer to the `Node` doc.

### PathToLeaf

`PathToLeaf` is a list of `ProofInnerNode`, where the list is ordered from
furthest to closet to the root node. 
For example, for the following path in the tree:

    Root
       \
        A
         \
          B
           \
            C
           
PathToLeaf will be storing nodes in the order of [C, B, A]

### ProofLeafNode

`ProofLeafNode` is similiar to `ProofInnerNode` where it holds few information about 
the leaf node without the extra paths information.

```golang
type ProofLeafNode struct {
	Key       cmn.HexBytes `json:"key"`
	ValueHash cmn.HexBytes `json:"value"`
	Version   int64        `json:"version"`
}
```

### RangeProof

Now we understand what each field type in a `RangeProof` means, we can now understand what a `RangeProof` is:

```golang
type RangeProof struct {
	LeftPath   PathToLeaf      `json:"left_path"`
	InnerNodes []PathToLeaf    `json:"inner_nodes"`
	Leaves     []ProofLeafNode `json:"leaves"`
}
```

Essentially a RangeProof represents a proof that a range of keys between either exists or absent
from the IAVL tree. This is important as RangeProof needs to support both verifying a single key (GetWithProof) and
list of keys (GetRangeWithProof).

`LeftPath` stores 
`InnerNodes` ...
`Leaves` ...

### Root Hash



## Proof Operations

Now we understand the fields that's part of the Range proof, we can start to understand what operations
are provided with these data.

The three operations a range proof provides are 1) Verify 2) VerifyItem 3) VerifyAbsence.

### Verify

One can verify a root hash and compares with the root hash that is computed from the RangeProof is equal.
As long as the tree content and the hashing mechanism remains the same, the root hash over a merkle tree should always be the same.
  
## VerifyItem

