/*
 * Copyright (c) 2019, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.cli.testing;

import java.util.Map;
import java.util.Random;
import com.oracle.coherence.common.base.Logger;
import com.tangosol.net.cache.CacheLoader;
import com.tangosol.net.cache.CacheStore;
import com.tangosol.util.Base;

/**
 * A simple {@link CacheLoader} implementation to demonstrate basic functionality.
 *
 * @author Tim Middleton 2020.02.17
 */
public class SimpleCacheStore
        extends SimpleCacheLoader
        implements CacheStore<Integer, String> {

    private final Random random = new Random();

    /**
     * Constructs a {@link SimpleCacheStore}.
     *
     * @param cacheName cache name
     */
    public SimpleCacheStore(String cacheName) {
        super(cacheName);
        Logger.info("SimpleCacheStore instantiated for cache " + cacheName);
    }

    @Override
    public void storeAll(Map<? extends Integer, ? extends String> mapEntries) {
        Logger.info("Store all size = " + mapEntries.size());
        CacheStore.super.storeAll(mapEntries);
    }

    @Override
    public void store(Integer key, String s) {
        Base.sleep(random.nextInt(3) + 1L);
    }

    @Override
    public void erase(Integer integer) {
        // noop
    }
}