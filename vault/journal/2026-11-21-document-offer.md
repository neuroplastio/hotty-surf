# 2026-11-21: a new block's document, offered to its Screen

Branch `doc-offer`, off main. Not pushed. Internal: no API change.
Step 2 of making new blocks in cells cheaper. Step 1, the parser's slabs
sized to the markup, is its own branch.

## Why

A new block in cells is a new Surface and a new Screen. Its document is
parsed twice. surf parses its render into a tree, renders the tree as
the document, and the Screen parses the document again
(html.ParseFragment in load). In plx's blocks the Screen's parse was all
the parsing a new view did, about 18 µs of it. plx's blocks link no
sheet, so the document is little more than the tree.

## What changed

- **`internal/rt.Host`.** A host in the program's own process that a
  surface's writer reaches: a `cells.Screen`, directly or through a
  wrapper of this module (`rt.Wrapper`, keyed by `rt.Seal`, so no app's
  writer is one). cells sets `rt.HostOf` in its init. A surface asks it
  once per Attach and keeps the answer.
- **The offer.** Document renders its nodes (the sheets, the style, the
  root) to the document as before. It then offers the host the name,
  the document and the nodes, lent for the call.
- **The copy, in cells, inside the call** (cells/offer.go):
  - the nodes go in one array and the attributes in another;
  - each node's attribute list is capped at its length, so an append
    moves it;
  - strings go in one buffer of the Screen's own, since a surf string
    slices its render;
  - atoms' names and namespaces stay static.
- **Used once.** The Screen takes the copy at the next `a=doc` only when
  its name and decoded payload are the offer's. Any `a=doc` or `a=del`
  drops it, and so does the next offer. Everything still goes on the
  wire, whole, so this changes no channel (P-3 doesn't apply).
- **The wrappers stay** (run's cellsHost, surftea's cellsOut, the dev
  report's devWriter). Each has a `Wrapped` method, so a refused delta
  is still noted where it was.

## Proof

- **The oracle** (cells/offer_test.go, `checked`) runs inside each
  offer, while the surface's tree is as rendered.
  - The Screen's copy (H) is the surface's tree (S) node for node, in
    attribute order.
  - H shares no node with S, no string but an atom's, and no attribute
    list's room.
  - The document's parse (W) is S as well. Where it isn't, the
    difference is noted, and the Screen shows S.
- **TestOfferIsTheTree.** Every story, with the kit's sheet and with
  none, takes 40 random steps: clicks, keys, the pointer, the wheel, and
  documents sent again, refused deltas included. Its Screen is held to
  a Screen that parses, cell for cell and caret for caret.
- **TestOfferEdges** covers:
  - an offer taken once;
  - the same document twice;
  - two documents before a write;
  - two surfaces on one Screen;
  - a tee, which gets no offer;
  - a compressed document;
  - other bytes than offered;
  - `a=del` and the delete of every surface;
  - a tree the surface changes after the call;
  - a detached surface.
- **FuzzHandoff** is seeded with FuzzParse's seeds and corpus and the
  stories.
- **TestOfferSparesTheParse** holds an offered new block to half a
  parsed one's allocations, and 4 KB fewer bytes.
- Each guarantee was sabotaged and caught:
  - strings shared;
  - attribute room left;
  - no name check;
  - an offer kept at `a=del`, or past a document it doesn't match;
  - no copy;
  - no look through wrappers.

## Found

- **x/net/html (v0.62) reorders attributes.** It puts a formatting
  element's attributes (an `a`, a `b`) in order of their names when it
  parses. The wire's tree then differs from surf's in attribute order
  alone (the breadcrumb and item stories). A DOM's attributes have no
  order that matters, so the oracle compares W's as a set and H's in
  order.

## Measured

Main a26e9fe against this branch before its rebase, fastest of four interleaved runs,
under the build cap, one CPU. The whole new block is a Surface, its
document on a new Screen, and a draw.

| block | main | doc-offer |
| --- | --- | --- |
| pill | 92.5 µs, 177,216 B, 672 allocs | 82.3 µs, 168,902 B, 408 allocs |
| line 1×120 | 113.2 µs, 208,163 B, 675 allocs | 103.2 µs, 200,795 B, 424 allocs |
| card 3×40 | 95.9 µs, 195,558 B, 633 allocs | 84.9 µs, 187,405 B, 394 allocs |

On main 44b002b, with step 1's slabs, at a load of 13 to 16 (so the
times are rough):

| block | main | doc-offer |
| --- | --- | --- |
| pill | 90.6 µs, 75,182 B, 672 allocs | 74.8 µs, 66,996 B, 408 allocs |
| line 1×120 | 109.9 µs, 106,141 B, 675 allocs | 90.4 µs, 98,901 B, 424 allocs |
| card 3×40 | 87.6 µs, 92,891 B, 633 allocs | 74.2 µs, 84,866 B, 394 allocs |

hello.wasm under TinyGo: 2,502,482 to 2,503,141 B.

A step of deltas on a Screen is unchanged (342,627 B against 342,676 B),
as deltas aren't offered yet. Most of a new block's bytes are still
surf's parser slab, which step 1 sizes: with both, the offered pill's
document and write take 14.5 KB where step 1 alone takes 20.7 KB.
