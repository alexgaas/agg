This article presents a comprehensive foundational framework for achieving effective data aggregation. The term **'effective'** 
implies the provision of:

The most efficient base algorithm or data structure tailored for: 
- Optimal latency / the best memory utilization in the given aggregation scenario.
- Maximal throughput in both parallel and distributed environments.
- Strategies for scaling performance both vertically and horizontally to accommodate the growing data flow.

_Additional notes_:
- Algorithmic or strategic considerations are presented with concise explanations and a list of pros and cons. 
- The goal is to provide the shortest yet most effective evaluation of each approach. I used simple words (such as
"slow", "simple", "complex", "terrible") to evaluate each approach.
- Details of evaluations (with benchmark numbers) could be found mostly in examples. 
All source examples are provided with **Golang** or **Java**. I had intent to not provide any benchmarks, but only
high level explanations to make this write-up as simple as possible. 
- Evaluation is given in appliance to aggregation operation. So if you see evaluation as `big overhead, terrible cache locality`,
that means in appliance to aggregation operation only.

**Parallel aggregation**
1. [One machine, One core](#one-machine-one-core)
   1. [Simple array](#simple-array)
   2. [Associative array approach definition](#associative-array-approach-definition) 
   3. [Binary Tree](#binary-tree)
   4. [Skip List](#skip-list)
   5. [Trie](#trie)
   6. [Lookup table](#lookup-table)
   7. [Hash map](#hash-map)
   8. [Trie + Hash map](#trie--hash-map)
2. [One machine, Multi-core](#one-machine-multi-core)
   1. [Hash map baseline](#hash-map-baseline)
   2. [Partitioning](#partitioning)
   3. [Parallel merge of hash maps](#parallel-merge-of-hash-maps)
   4. [Ordered merge of hash maps](#ordered-merge-of-hash-maps)
   5. [Concurrent hash tables](#concurrent-hash-tables)
      1. [Baseline - shared hash map with mutex](#baseline---shared-hash-map-with-mutex)
      2. [A set of small hash tables, each with its own mutex.](#a-set-of-small-hash-tables-each-with-its-own-mutex)
      3. [Shared hash table with a spin-lock mechanism on each cell](#shared-hash-table-with-a-spin-lock-mechanism-on-each-cell)
      4. [Lock-free hash table](#lock-free-hash-table)
   6. [Shared hast table + thread local hash tables](#shared-hast-table--thread-local-hash-tables)
   7. [Two level hash table](#two-level-hash-table)

**Distributed aggregation**
1. [Baseline (trivial way)](#baseline-trivial-way)
2. [Ordered merge](#ordered-merge)
3. [Partitioned merge](#partitioned-merge)
4. [Reshuffle + Partitioned merge](#reshuffle--partitioned-merge)

# Parallel aggregation

## One machine, one core
The simplest approach, though often not the most efficient, would be to use an array as the base structure to aggregate.
Let's consider this case:
### Simple array
- Read the data and store it in an array.
- Sort the data by key.
- As a result of the sorting operation, groups of data with the same key will be placed consecutively.
- Iterate through the array by groups of keys and perform aggregate functions.
#### Pros:
+ Very simple interface of aggregate function.
+ Aggregate functions may be implemented efficient way.
+ You can run any scripts to reduce in streaming mode since data sorted out.
#### Cons:
Let's define N as number of data rows and M as number of keys. So if `N > M` (usual case, as example -  group by operating system and count popularity):
- Slow / bad runtime.
- We spend `O(N)` of memory to sort whole dataset, not `O(M)` of keys.
#### Example
See example in `golang/group/onecore/simple_array`

### Associative array approach definition
Better way to aggregate to use associative array (get some value by key):

`key tuple -> states of aggregate function` (so for tuple we're getting group by for we assign aggregate function)

Then we iterate through data rows, looking at key, pull out state of aggregate function, based on key,
and update that state.

Which associative array we may use:

- Hash map / lookup table.
- Binary tree (skip list, b-tree).
- Trie (or trie + hash map).

### Binary Tree
#### Cons:
- Each element in the binary tree incurs a notable overhead with three pointers, totaling 24 bytes per element.
- Terrible cache locality.
- _As outcome_ - slow runtime / big memory consumption.

### Skip list
#### Pros:
+ In scenarios where a list needs to be consistently kept in memory with concurrent access, a skip list may prove to be a fitting solution.
#### Cons:
- Same issues as for binary tree - big overhead, terrible cache locality.

### Trie
#### Cons:
- Trie might be compact, but then we have no chance to update it.
- Or (if we need to update) trie will require a huge amount of memory (to update).

### Lookup table
#### Pros:
+ It's an ideal fit when you need to aggregate numeric keys, especially if they are limited to no more than ~16 bits.
#### Cons:
- Does not work for any other scenario like string, etc.

### Hash map:
#### Pros:
+ Have the best efficiency by runtime and memory.
#### Cons:
- Many implementation details - which memory layout, which mechanism of solving hash collisions,
  which hash function, how to make hash map for tuple, how to make hash map for string key of variable length.
#### Example
See example in `golang/group/onecore/hashmap`

### Trie + Hash map
We can bitwise trie and for each first bit of key assign each own hash map.

Example in progress ...

## One machine, multi-core

### Hash map baseline
As baseline let's make:
+ Different threads read different chunks of data by demand.
+ Aggregate data independently in their local hash maps.
+ When all data around all threads have been aggregated, we need to merge them.
+ As simple merge algorithm we can just iterate through all tables instead of first and move all data to first one.
+ As small improvement we can have primary table as biggest one.

As outcome:
- Phase of reading and preliminary aggregation have parallelized.
- Phase of merging going sequentially.

#### Pros:
+ Simple.
+ It demonstrates scalability in scenarios with a small number of keys post-aggregation phase, 
particularly with a low cardinality in the 'group by' operation. In such cases, the second phase costs are minimal, 
allowing for efficient parallelization.
#### Cons:
- Not scalable with big cardinality of group by.

_Explanation_: Let's define N as number of data rows and M as number of keys.
Let's define N as number of data rows and M as number of keys. `O(M)` of work made sequentially since
if we have big M (what is cardinality of group by), work can't be parallelized.

### Partitioning
Let's split whole dataset for approximately equal data blocks. For each data block let's make aggregation in two phases:
#### Phase 1
Different number of threads are going to process different parts of data blocks (which can take and process first, there is
no any contention or synchronization here). In the thread by using separate simple hash function we hash key into thread number
and remember it: 
`hash: key -> bucket_num`
#### Phase 2
Each thread iterates through data block and takes for aggregation only rows with appropriate bucket number.

_As minor improvement_: we can implement all as one phase - then every thread calculates hash function from all strings
every time, it works if it's cheap to do in terms of runtime.

#### Pros:
+ Good scalability with big cardinality and evenly distribution of keys.
+ Simple design.
#### Cons
- If data distributed not evenly by keys, Phase 2 won't scale.
  That's typical case actually in real life - look on the key distribution in `phones_data.csv`:
  most of the keys by OS as example _Android_, then _iOS_, then a bit of windows and others.
  Data volume at real life every time distributed by power low
  (there are keys with many data and there are keys with very little data volume).
  So in this algorithm one key with big data volume will be served by one thread and accordingly
  won't scale well. Look on production-ready case here - Hadoop or any other map-reduce system, this issue called squid case
  over there.

_More cons_:
- if data block is small we get small granularity of threads (if many threads trying to solve such a small problems
  we're getting more overhead for thread creation than scale); that also brings more overhead for synchronization
- if data block size is huge we're getting bad cache locality
- on Phase 2, memory bandwidths (part of it at least) will multiply on the number of threads
- you need additional hash function independent of that which in hash table

#### Example
See example in `golang/group/multicore/partitioning`

### Parallel merge of hash maps
Let's back to our hashmap baseline. In that case we did not scale Phase 2 - merge of hash maps.
Could we make that phase parallel?
#### Approach 1
Let's combine both baseline hashmap approach and partitioning approach:
+ Run threads with local hashmaps using buckets.
+ As outcome each thread will return hash map with different keys.
+ Sequentially merge hash maps to new one assuming there is no any costs b/c keys different for each hash map.
#### Cons:
- In this model, each thread is responsible for processing the entire dataset, with the scalability constrained by the 
shared RAM bandwidth across all threads. This limitation persists even when an ample number of CPU resources are available.
#### Approach 2
+ Let's resize hash maps gotten from threads to one size.
+ Split them implicitly on different sets of keys.
+ In different threads we're going to merge appropriate sets of keys.
#### Cons:
Extremely complicated code because we need to resolve problem of collision resolution chains in the start and end of new 
hash map during the process of parallel merge.
#### Example
Still in progress...

### Ordered merge of hash maps
Data in any hash map are (almost) ordered by reminder of division of hash function on the size of hash map.

Still in progress...

### Concurrent hash tables

#### Baseline - shared hash map with mutex
#### Pros:
+ Simple.
#### Cons:
- Negative scalability - more threads we have more negative scalability.

#### A set of small hash tables, each with its own mutex
Let's make N mutexes. We have simple hash function to define number of buckets. Every bucket is
protected by mutex to prevent out of sync state in the bucket when it is updated by number of threads.
#### Pros:
+ If data distributed evenly for some reason that approach will scale.
#### Cons:
- Since data never distributed evenly (but _usually distributed by power law_) we will get contention on hot bucket, 
so it won't scale.

#### Shared hash table with a spin-lock mechanism on each cell
#### Cons:
- Because the OS scheduler is unaware of spin locks, it may switch to another thread, 
potentially causing your code to linger in the top CPU percentile without progress.
- You're having same issue with contention on hot cell since aggregating data never distributed evenly.

#### Lock-free hash table
#### Cons:
- Hard to resize. They not resizable at all or having extremely complicated code which in addition will be slow.
- Lock-free means synchronization even if it is lock free. Best way in terms of scalability to avoid any synchronization.

### Shared hast table + thread local hash tables
Let's make one shared hash table with mutex on the cell. If cell already is locked we put data to local hash table.
Then all hot cells (cells with contention on it) will be placed in local hash tables. As outcome highly likely all local 
hash tables going to be small. In the end we merge all local hash tables to the global one - this phase should not take too long since
local hash tables must be tiny in match to global.

Possible improvements:
- Look first into local hash table for key.
- If chain to resolve collision on the shared hash table reaching N put on local instead global.

#### Pros:
+ Great scalability.
+ Simple design.
#### Cons:
- Many lookup operations / many instructions - more slowness.
#### Example
See example in `golang/group/multicore/global_local_hashmap`

### Two level hash table
#### Phase 1 (distribute phase)
In each thread independently let's make associative array of `num_buckets` with hash table for each element.
We have constant of num_buckets as 256 and same number of hash tables accordingly:
`num_threads * num_buckets of hash tables`
Number of bucket defined by different simple hash function.

As outcome, we have matrix of hash tables:
```
\/hash tables
    1|2|3|...|10 - threads / tables
1   . . .      .
2   . . .      .
3   . . .      .
...  
256 . . .      .
```

#### Phase 2 (merge)
On this phase we merge `num_threads * num_buckets of hash tables` in the same `num_bickets` of hash tables,
making parallel merge by buckets very natural way.
#### Pros:
+ Excellent scalability.
+ Simple design.
+ All data in the end divided on partitions. That's key advantage if you're doing distributed grouping later between network nodes.
#### Cons
- If we have small cardinality of group by we spend too much of memory to allocation so many hash tables.
#### Example
See example in `golang/group/multicore/two_level_hashmap`

---
# Distributed aggregation
One machine is having shared memory, which can be used by N threads. If we need
to manage data on different machines, we do not have any shared memory. As outcome:

- there is no option to use work-stealing algorithm
- data will be transferred by network

## Baseline (trivial way)

Let's send intermediate results to server initiator of query from data nodes (clients). Sequentially
put all results to one hash table.

#### Pros:
+ simple
+ good scalability with small cardinality of group by

#### Cons:
- no scalability with big cardinality
- you need to get as much memory as much data coming from data nodes (in fact you need memory of all transferred data)

#### Example
See example in `golang/dist-group/baseline/`

## Ordered merge

Let's send intermediate results to server initiator of query from data nodes in *defined order* (that means
data must be sorted out on data nodes same and known by server initiator algorithm).
Then we can pull them in parallel by some chunks to server and merge sorted threads.

#### Pros:
+ simple
+ you spend O(1) memory on merge

#### Cons:
- merge (aggregation itself) is sequential so no scalability with big cardinality of keys
- merge of sorted thread in heap is slow itself
- you need sort out data on servers or use fancy algorithms (robinhood tables)

#### Example
See example in `golang/dist-group/ordered-merge/`

## Partitioned merge

Let's send intermediate results to server initiator of query from data nodes divided by separate
consistent buckets-partitions in conformed order.
As result, we can merge by one or few buckets in parallel.

#### Pros:
+ We spend `num_bucket` less memory, then size of result.
  We can merge by one partition or 16 in parallel depends on our memory strategy.
+ As outcome of first ^ - we can easily make parallel merge of N buckets - that have great scalability.

#### Cons:
- Phase 2 not scaling by servers in network. Merge happens only on one server initiator of query

#### Example
See example in `golang/dist-group/partitioned_merge/`

## Reshuffle + Partitioned merge

We have to scale phase of merge between servers not just by cores of one server initiator.

On the data nodes, we obtain intermediate results in the form of partitions.
These partitions are then transferred between nodes in a way that ensures each node receives different partitions,
ensuring that the partitioned data remains unique per node.

Then we can use N servers-initiators to merge data in parallel (plus each server can use M cores additionally).

#### Pros:
+ Great scalability distributed between N machines in the network
+ Since data not overlap each other by buckets we can just a store data locally on nodes and
  have result as distributed table on the cluster

#### Cons:
- Complex coordination between data nodes

#### Example
In progress...