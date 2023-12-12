package org.alexgaas;

import java.util.Map;

public interface SharedMap<K,V> extends Map<K, V> {
    SharedMap<K, V> with(K key, V value);

    SharedMap<K,V> without(K key);
}