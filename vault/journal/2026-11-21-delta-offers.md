# 2026-11-21: a delta's markup, offered to its Screen

Branch `delta-offer`, off `doc-offer`. Not pushed. Internal: no API
change. Step 3 of making new blocks in cells cheaper, after the
document offer (2026-11-21-document-offer.md).

## Why

The document offer spared a new block's second parse. Every delta with
markup (an inner, a replace, a before, an append) was still parsed by
the Screen from the bytes surf had just rendered from its own nodes.

## What changed

- **`rt.Host.OfferDelta(name, frags)`.** Each `rt.Fragment` is an op's
  markup and the nodes it renders, in the order of the ops that carry
  markup. surf's differ collects them as it renders (`render`, `inner`)
  only where the surface has a host. The surface offers them as it
  writes the delta, lent for the call, as the document's nodes are.
- **The copy, in cells, inside the call.** Every fragment's nodes are
  copied in one `copyNodes`. Each fragment's list is capped at its
  length.
- **Taken by bytes.** The next `a=delta` for the surface takes the
  offer. Each op with markup (`fragment`) takes the next fragment's copy
  where its markup is that fragment's, byte for byte. From the first op
  of other bytes, the rest of the delta is parsed.
  - Any `a=delta` ends the offer, another surface's too, and so do
    `a=doc`, `a=del` and the next offer.
  - The delta stays on the wire whole, so no channel changes (P-3
    doesn't apply).

## Proof

- **The oracle** (`checked.OfferDelta`) checks each fragment's copy.
  It must be the surface's nodes, sharing nothing, with no links, and
  with no room for the next fragment's. The markup's parse under the
  nodes' parent must be those nodes too, or else the difference is
  noted. None was.
- **TestDeltaOffersAreTheTree.** 60 random documents each take 30
  steps: another document's markup, a part added, or random texts and
  attributes. That makes every op the differ writes. The Screen that
  takes offers is held to one that parses, cell for cell. 6,147
  fragments were offered, and the Screen's trees took 17,812 of their
  nodes.
- **TestOfferIsTheTree** (the stories' walks) counts the fragments
  taken as well.
- **TestDeltaOfferEdges** covers:
  - fragments taken in order past ops without markup;
  - the same delta twice;
  - other bytes, after which a fragment's bytes are parsed too;
  - another surface's delta and another surface's offer;
  - a document between, and `a=del` and the delete of every surface;
  - a compressed delta;
  - nodes changed after the call;
  - one array for every fragment's copies.
- **FuzzHandoff** takes two renders now: a document, then the delta to
  the next.
- **TestDeltaOfferSparesTheParse.** Rows that come and go take 395 → 253
  allocations and 63.8 → 52.1 KB a step.
- Each guarantee was sabotaged and caught:
  - no markup check;
  - no stop at other bytes;
  - no name check;
  - an offer kept past another surface's delta, a document or `a=del`;
  - room left for the next fragment;
  - no copy.

## Measured

Main 44b002b, doc-offer and this branch, fastest of four interleaved
runs, under the build cap, at a load of 11 to 13 (so the times are
rough). The bench step is 12 rows written again, on a Screen.

| bench | main | doc-offer | delta-offer |
| --- | --- | --- | --- |
| step on a Screen | 254,750 B, 1,274 allocs | 254,577 B, 1,274 | 234,162 B, 1,170 |
| new pill | 75,182 B, 672 allocs | 66,996 B, 408 | 66,996 B, 408 |

hello.wasm under TinyGo: 2,503,141 to 2,504,013 B.
