package org.alexgaas;

import java.io.Externalizable;
import java.io.IOException;
import java.io.ObjectInput;
import java.io.ObjectOutput;
import java.lang.reflect.Array;
import java.util.*;

public class HashTrieMap<K,V> extends AbstractMap<K,V> implements SharedMap<K,V>, Externalizable {
    private static final long serialVersionUID = -976712368400781259L;
    private static final int HASH_SHIFT = 5;
    private static final int HASH_MASK = (1 << HASH_SHIFT) - 1;

    // A node in the trie structure, which has a location-defined hash code prefix
    // and contains a list of children which may be nodes, a single entry, or a collision
    // node.
    // For a chunk 'h' of hash code (which is in the range [0 HASH_MASK]), it is present if
    // hasChild & (1 << h) != 0.
    // The index of the child of this 'h' is popCount(hasChild & ((1 << h) - 1)).
    private static class Node {
        // children :: [Node | SimpleMapEntry | SimpleMapEntry[]]
        public final Object[] children;
        public final int hasChild;
        private Node(Object[] children, int hasChild) {
            this.children = children;
            this.hasChild = hasChild;
            assert children != null;
            assert Integer.bitCount(hasChild) == children.length;
        }
    }

    // these would all be final, but for Java's horrid readExternal() deserialization
    // root :: Node | SimpleMapEntry | SimpleMapEntry[] | Null
    private Object mRoot;
    private int mSize;
    // cached implementations
    private transient Set<Entry<K, V>> mEntrySet = null;
    private transient Set<K> mKeySet = null;

    private HashTrieMap(Object root, int size) {
        mRoot = root;
        mSize = size;
    }

    // *** Factories ***

    public HashTrieMap() {
        this(null, 0);
    }
    public HashTrieMap(Map<? extends K, ? extends V> m) {
        HashTrieMap<K,V> hashMap;
        if (m instanceof HashTrieMap) {
            // O(1) copy - we can just view the same data
            hashMap = (HashTrieMap) m;
        } else {
            // O(n log(n)) copy from a general Map
            hashMap = empty();
            for (Map.Entry<? extends K, ? extends V> entry : m.entrySet()) {
                hashMap = hashMap.with(entry.getKey(), entry.getValue());
            }
        }
        mRoot = hashMap.mRoot;
        mSize = hashMap.mSize;
    }
    public static final HashTrieMap EMPTY = new HashTrieMap(null, 0);
    public static <K,V> HashTrieMap<K,V> empty() {
        return EMPTY;
    }
    public static <K,V> HashTrieMap<K,V> singleton(K key, V value) {
        return new HashTrieMap<K,V>(new SimpleImmutableEntry<K,V>(key, value), 1);
    }
    public static <K,V> HashTrieMap<K,V> of() {
        return EMPTY;
    }

    public static <K,V> HashTrieMap<K,V> of(K key, V value, Object... keyValues) {
        if (keyValues.length % 2 != 0) {
            throw new IllegalArgumentException("HashTrieMap.of() called with an odd number of keyValues (cannot partition them into pairs)");
        }
        HashTrieMap<K,V> m = singleton(key, value);
        for (int i = 0; i < keyValues.length; i += 2) {
            m = m.with((K) keyValues[i], (V) keyValues[i + 1]);
        }
        return m;
    }

    // *** AbstractMap ***

    private class EntrySet extends AbstractSet<Entry<K, V>> {
        @Override
        public Iterator<Entry<K, V>> iterator() {
            return new PreOrderIterator<K, V>(HashTrieMap.this.mRoot);
        }

        @Override
        public int size() {
            return HashTrieMap.this.size();
        }

        // overridden for performance
        @Override
        public boolean contains(Object o) {
            Entry<K,V> mapping = (Entry) o;
            K key = mapping.getKey();
            V value = mapping.getValue();
            return key != null && value != null && value.equals(HashTrieMap.this.get(key));
        }
    }

    private class KeySet extends AbstractSet<K> {
        @Override
        public Iterator<K> iterator() {
            final Iterator<Entry<K,V>> entryIterator = new PreOrderIterator<K, V>(HashTrieMap.this.mRoot);
            return new Iterator<K>() {
                @Override
                public boolean hasNext() {
                    return entryIterator.hasNext();
                }
                @Override
                public K next() {
                    return entryIterator.next().getKey();
                }
                @Override
                public void remove() {
                    throw new UnsupportedOperationException("remove() called on immutable iterator (you cannot mutate a SharedMap using its' iterator)");
                }
            };
        }

        @Override
        public int size() {
            return HashTrieMap.this.size();
        }

        // overridden for performance
        @Override
        public boolean contains(Object key) {
            return HashTrieMap.this.get(key) != null;
        }
    }

    @Override
    public Set<Entry<K, V>> entrySet() {
        Set<Entry<K, V>> s = mEntrySet;
        return s != null ? s : (mEntrySet = new EntrySet());
    }

    // overridden for performance
    @Override
    public Set<K> keySet() {
        Set<K> s = mKeySet;
        return s != null ? s : (mKeySet = new KeySet());
    }

    // overridden for performance
    @Override
    public boolean containsKey(Object key) {
        return get(key) != null;
    }

    // *** SharedMap ***

    @Override
    public int size() {
        return mSize;
    }

    @Override
    public V get(Object key) {
        if (key == null || mRoot == null) {
            return null;
        }

        final int hash = key.hashCode();
        int shift = 0;
        Object current = mRoot;
        while (true) {
            if (current instanceof Node) {
                // Case 1: a node - continue searching from the child node
                Node currentNode = (Node) current;
                // take a 5-bit chunk of the hash code
                int offset = (hash >>> shift) & HASH_MASK;
                int mask = 1 << offset;
                if ((currentNode.hasChild & mask) != 0) {
                    // we have found a child (which must be non-null), so continue searching
                    current = currentNode.children[Integer.bitCount(currentNode.hasChild & (mask - 1))];
                    shift += HASH_SHIFT;
                } else {
                    // no child with that hash - key must be missing
                    return null;
                }

            } else if (current instanceof SimpleImmutableEntry) {
                // Case 2: an Entry - return the value only if the key matches
                SimpleImmutableEntry<K, V> entry = (SimpleImmutableEntry) current;
                return key.equals(entry.getKey()) ? entry.getValue() : null;

            } else {
                // Case 3: must be a collision (Entry[]) - do a linear search in the collision 'array map'
                for (SimpleImmutableEntry<K,V> entry : (SimpleImmutableEntry[]) current) {
                    if (key.equals(entry.getKey())) {
                        return entry.getValue();
                    }
                }
                return null;
            }
        }
    }

    // helper function to implement 'with'
    // connects up the object 'next' to the trie between 'root' & 'parent'
    // returns the new value of the trie root
    private static Object connect(Object root, Object[] parent, int parentIndex, Object next) {
        if (root == null) {
            // we're looking at the root node, return it
            return next;
        } else {
            // we're looking at some child node, the root doesn't change,
            // but we must connect 'next' up to its' parent instead
            parent[parentIndex] = next;
            return root;
        }
    }

    private static <K> int findCollision(SimpleImmutableEntry<K,?>[] entries, K key) {
        int index = 0;
        while (index < entries.length) {
            if (key.equals(entries[index].getKey())) {
                break;
            }
            ++index;
        }
        return index;
    }

    private static <T> T[] copyWithout(Class<T> tClass, T[] original, int index) {
        T[] result = (T[]) Array.newInstance(tClass, original.length - 1);
        for (int i = 0; i < result.length; ++i) {
            result[i] = original[i + (index <= i ? 1 : 0)];
        }
        return result;
    }

    @Override
    public HashTrieMap<K, V> with(K key, V value) {
        if (key == null) {
            throw new NullPointerException("Cannot add a null key to a HashTrieMap");
        }
        if (value == null) {
            throw new NullPointerException("Cannot add a null value to a HashTrieMap");
        }
        // special case the root - the only nullable object
        if (mRoot == null) {
            return singleton(key, value);
        }

        // convenience
        final SimpleImmutableEntry<K, V> entry = new SimpleImmutableEntry<K,V>(key, value);
        final int hash = key.hashCode();

        // a lot of state!
        Object newRoot = null;
        Object[] parent = null;
        int parentIndex = -1;
        Object current = mRoot;
        int shift = 0;
        while (true) {
            // are we at a leaf (single-element or multi-element 'collision')
            if (current instanceof SimpleImmutableEntry) {
                SimpleImmutableEntry<K,V> currentEntry = (SimpleImmutableEntry) current;

                if (key.equals(currentEntry.getKey())) {
                    // TERMINAL replace the existing entry (same key)
                    return new HashTrieMap<K,V>(connect(newRoot, parent, parentIndex, entry), mSize);

                } else if (shift < Integer.SIZE) {
                    // split into a Node (and get handled as 'instanceof Node' below)
                    int currentEntryHash = (currentEntry.getKey().hashCode() >>> shift) & HASH_MASK;
                    current = new Node(new Object[] { currentEntry }, 1 << currentEntryHash);

                } else {
                    // TERMINAL generate a 2-element collision array
                    return new HashTrieMap<K,V>(connect(newRoot, parent, parentIndex, new SimpleImmutableEntry[] { entry, currentEntry }), mSize + 1);
                }

            } else if (current instanceof SimpleImmutableEntry[]) {
                // a collision - we must be a leaf, so just add or replace the entry in the collision list
                SimpleImmutableEntry<K,V>[] currentCollision = (SimpleImmutableEntry[]) current;

                // find the existing index of the match
                int idx = findCollision(currentCollision, key);
                boolean found = idx < currentCollision.length;
                // expand if needed, and write in the new element
                SimpleImmutableEntry<K,V>[] newCollision = Arrays.copyOf(currentCollision, currentCollision.length + (found ? 0 : 1));
                newCollision[idx] = entry;

                // TERMINAL
                return new HashTrieMap<K,V>(connect(newRoot, parent, parentIndex, newCollision), mSize + (found ? 0 : 1));
            }

            // then we must be at a Node
            Node currentNode = (Node) current;
            // take a 5-bit chunk of the hash code
            int offset = (hash >>> shift) & HASH_MASK;
            int mask = 1 << offset;
            int childIndex = Integer.bitCount(currentNode.hasChild & (mask - 1));
            if ((currentNode.hasChild & mask) == 0) {
                // missing key - expand space & add the entry into the empty slot
                Object[] children = new Object[currentNode.children.length + 1];
                for (int i = 0; i < childIndex; ++i) {
                    children[i] = currentNode.children[i];
                }
                children[childIndex] = entry;
                for (int i = childIndex + 1; i < children.length; ++i) {
                    children[i] = currentNode.children[i-1];
                }
                // TERMINAL
                return new HashTrieMap<K,V>(connect(newRoot, parent, parentIndex, new Node(children, currentNode.hasChild | mask)), mSize + 1);

            } else {
                // NON-TERMINAL hash prefix collision
                current = currentNode.children[childIndex];
                shift += HASH_SHIFT;
                Object[] children = Arrays.copyOf(currentNode.children, currentNode.children.length);
                children[childIndex] = null;
                newRoot = connect(newRoot, parent, parentIndex, new Node(children, currentNode.hasChild));
                parent = children;
                parentIndex = childIndex;
                //continue; (implicit)
            }
        }
    }

    private static <K> Object removeFrom(Object current, K key, int hash, int shift) {
        if (current instanceof Node) {
            Node currentNode = (Node) current;
            // take a 5-bit chunk of the hash code
            int offset = (hash >>> shift) & HASH_MASK;
            int mask = 1 << offset;
            if ((currentNode.hasChild & mask) == 0) {
                // key not found - don't modify
                return current;

            } else {
                int childIndex = Integer.bitCount(currentNode.hasChild & (mask - 1));
                Object currentChild = currentNode.children[childIndex];
                Object newChild = removeFrom(currentChild, key, hash, shift + HASH_SHIFT);
                if (currentChild == newChild) {
                    // key not found (recursively) - don't modify
                    return current;

                } else if (newChild == null) {
                    if (currentNode.children.length == 1) {
                        // no children left - delete this node
                        return null;

                    } else if (currentNode.children.length == 2
                            && !(currentNode.children[1 - childIndex] instanceof Node)) {
                        // can 'collapse' to an Entry for the other child, as long as it isn't a Node
                        return currentNode.children[1 - childIndex];

                    } else {
                        // remove the child
                        return new Node(copyWithout(Object.class, currentNode.children, childIndex), currentNode.hasChild & ~mask);
                    }

                } else {
                    // rebuild children map
                    Object[] newChildren = Arrays.copyOf(currentNode.children, currentNode.children.length);
                    newChildren[childIndex] = newChild;
                    return new Node(newChildren, currentNode.hasChild);
                }
            }

        } else if (current instanceof SimpleImmutableEntry) {
            // if current is a match, remove it by returning null, otherwise don't modify
            return key.equals(((SimpleImmutableEntry) current).getKey()) ? null : current;

        } else { // must be an Entry[]
            SimpleImmutableEntry<K,?>[] currentCollision = (SimpleImmutableEntry[]) current;
            int idx = findCollision(currentCollision, key);

            if (currentCollision.length <= idx) {
                // key not found - don't modify
                return current;

            } else if (currentCollision.length == 2) {
                // only one left - not a collision list anymore!
                return currentCollision[1 - idx];

            } else {
                // create a new array without this entry
                return copyWithout(SimpleImmutableEntry.class, currentCollision, idx);
            }
        }
    }

    @Override
    public HashTrieMap<K, V> without(K key) {
        if (key == null) {
            throw new NullPointerException("Cannot add a null key to a HashTrieMap");
        }
        if (mRoot == null) {
            return this; // we were empty - still empty
        }
        // delegate to a recursive helper
        Object newRoot = removeFrom(mRoot, key, key.hashCode(), 0);
        return newRoot == mRoot ? this : new HashTrieMap<K, V>(newRoot, mSize - 1);
    }

    private static class PreOrderIterator<K,V> implements Iterator<Entry<K,V>> {
        // stack of nodes from root to current leaf
        private static final int MAX_DEPTH = 7; // max size of the deque is 32 bits / 5 (bits/node) = 7 nodes
        private final Node[] mNodeStack = new Node[MAX_DEPTH];
        private final int[] mNodeIndexStack = new int[MAX_DEPTH];
        private int mNodeStackPointer = -1;
        // the current leaf (and position in the collision array, if needed)
        private Object mCurrent;
        private int mCurrentCollisionIndex = -1;

        private PreOrderIterator(Object root) {
            if (root instanceof Node) {
                mNodeStack[0] = (Node) root;
                mNodeIndexStack[0] = -1;
                mNodeStackPointer = 0;
                moveToNext();
            } else {
                mCurrent = root;
            }
        }

        private void moveToNext() {
            if (mCurrent instanceof SimpleImmutableEntry[]
                    && ++mCurrentCollisionIndex < ((SimpleImmutableEntry[]) mCurrent).length) {
                // do nothing - we've already advanced to the next collision node

            } else if (mNodeStackPointer < 0) {
                // this can happen in a singleton set - where the node stack is empty even though we have an element
                mCurrent = null;

            } else {
                // walk up the tree until we have successfully advanced to the next child
                while (mNodeStack[mNodeStackPointer].children.length <= ++mNodeIndexStack[mNodeStackPointer]) {
                    --mNodeStackPointer;
                    if (mNodeStackPointer < 0) {
                        mCurrent = null;
                        return;
                    }
                }
                // walk down the tree until we have found the first child from the current top of the stack
                while (true) {
                    Object child = mNodeStack[mNodeStackPointer].children[mNodeIndexStack[mNodeStackPointer]];
                    if (child instanceof SimpleImmutableEntry) {
                        mCurrent = child;
                        return;

                    } else if (child instanceof SimpleImmutableEntry[]) {
                        assert(0 < ((SimpleImmutableEntry[]) child).length);
                        mCurrent = child;
                        mCurrentCollisionIndex = 0;
                        return;

                    } else { // Node
                        // push onto the stack & keep searching
                        assert(0 < ((Node) child).children.length);
                        ++mNodeStackPointer;
                        mNodeStack[mNodeStackPointer] = (Node) child;
                        mNodeIndexStack[mNodeStackPointer] = 0;
                    }
                }
            }
        }

        @Override
        public boolean hasNext() {
            return mCurrent != null;
        }

        @Override
        public Map.Entry<K, V> next() {
            Map.Entry<K,V> next =
                    (mCurrent instanceof SimpleImmutableEntry[])
                    ? ((SimpleImmutableEntry[]) mCurrent)[mCurrentCollisionIndex]
                    : (SimpleImmutableEntry) mCurrent;
            moveToNext();
            return next;
        }

        @Override
        public void remove() {
            throw new UnsupportedOperationException("remove() called on immutable iterator (you cannot mutate a SharedMap using its iterator)");
        }
    }

    @Override
    public void writeExternal(ObjectOutput out) throws IOException {
        out.writeInt(mSize);
        for (Map.Entry<K,V> entry : this.entrySet()) {
            out.writeObject(entry);
        }
    }

    @Override
    public void readExternal(ObjectInput in) throws IOException, ClassNotFoundException {
        mSize = in.readInt();
        HashTrieMap<K,V> m = HashTrieMap.empty();
        for (int i = 0; i < mSize; ++i) {
            Map.Entry<K,V> entry = (Map.Entry<K,V>) in.readObject();
            m = m.with(entry.getKey(), entry.getValue());
        }
        mRoot = m.mRoot;
    }
}