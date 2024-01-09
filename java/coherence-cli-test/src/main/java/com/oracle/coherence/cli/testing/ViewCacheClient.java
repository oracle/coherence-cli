/*
 * Copyright (c) 2024 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.cli.testing;

import com.tangosol.net.CacheFactory;
import com.tangosol.net.NamedCache;
import com.tangosol.util.Base;

/**
 * Aview cache client.
 *
 * @author tam 2024.01.07
 */
public class ViewCacheClient {

    /**
     * Private constructor.
     */
    private ViewCacheClient() {
    }

    /**
     * Program entry point.
     *
     * @param args the program command line arguments
     */
    public static void main(String[] args) {
        NamedCache<Integer, String> view1 = null;
        NamedCache<Integer, String> view2 = null;
        try {
            System.out.println("Sleeping 30 seconds");
            Base.sleep(30_000L);
            view1 = CacheFactory.getCache("view-cache-1");
            view2 = CacheFactory.getCache("view-cache-2");
            RestServer.populateViewCache(null);
            System.out.println("Populated");
        }
        catch (Throwable thrown) {
            throw new RuntimeException("Failed to start view cache client", thrown);
        }

        while (true) {
            System.out.println(view1.size());
            System.out.println(view2.size());
            Base.sleep(5000L);
        }

    }
}
