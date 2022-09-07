/*
 * Copyright (c) 2019, 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.cli.testing;

import java.io.IOException;
import java.io.OutputStream;

import java.lang.reflect.Method;

import java.net.InetSocketAddress;

import java.util.HashMap;
import java.util.Map;
import java.util.stream.Collectors;

import com.tangosol.net.CacheFactory;
import com.tangosol.net.Cluster;
import com.tangosol.net.DefaultCacheServer;
import com.tangosol.net.NamedCache;

import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

/**
 * A simple Http server that is deployed into a Coherence cluster
 * and can be used to perform various tests.
 *
 * @author jk  2019.08.09
 */
public class RestServer {

    /**
     * Private constructor.
     */
    private RestServer() {
    }

    /**
     * Program entry point.
     *
     * @param args the program command line arguments
     */
    public static void main(String[] args) {
        try {
            int        port   = Integer.parseInt(System.getProperty("test.rest.port", "8080"));
            HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);

            server.createContext("/ready", RestServer::ready);
            server.createContext("/env", RestServer::env);
            server.createContext("/props", RestServer::props);
            server.createContext("/suspend", RestServer::suspend);
            server.createContext("/resume", RestServer::resume);
            server.createContext("/populate", RestServer::populate);
            server.createContext("/populateFlash", RestServer::populateFlash);
            server.createContext("/populateRam", RestServer::populateRam);
            server.createContext("/populateFederation", RestServer::populateFederation);
            server.createContext("/edition", RestServer::edition);
            server.createContext("/version", RestServer::version);
            server.createContext("/registerMBeans", RestServer::registerMBeans);
            server.createContext("/executorPresent", RestServer::isExecutorPresent);
            server.createContext("/healthPresent", RestServer::isHealthCheckPresent);

            server.setExecutor(null); // creates a default executor
            server.start();

            System.out.println("REST server is UP! http://localhost:" + server.getAddress().getPort());

            // if the executor is present we need to run Coherence.main to start executor
            if (canFindExecutor()) {
                try {
                    Class<?>   clazz        = Class.forName("com.tangosol.net.Coherence");
                    Class<?>[] argumentType = new Class[] {String[].class};
                    Method     mainMethod   = clazz.getMethod("main", argumentType);
                    System.err.println("Found Coherence " + clazz.getName());
                    mainMethod.invoke(null, (Object) new String[] {});
                }
                catch (Exception e) {
                    // ignore
                }
            }

        }
        catch (Throwable thrown) {
            throw new RuntimeException("Failed to start http server", thrown);
        }

        DefaultCacheServer.main(args);
    }

    private static void send(HttpExchange t, int status, String body) throws IOException {
        t.sendResponseHeaders(status, body.length());
        OutputStream os = t.getResponseBody();
        os.write(body.getBytes());
        os.close();
    }

    private static void ready(HttpExchange t) throws IOException {
        send(t, 200, "OK");
    }

    private static void env(HttpExchange t) throws IOException {
        String data = System.getenv()
                            .entrySet()
                            .stream()
                            .map(e->String.format("{\"%s\":\"%s\"}", e.getKey(), e.getValue()))
                            .collect(Collectors.joining(",\n"));

        send(t, 200, "[" + data + "]");
    }

    private static void props(HttpExchange t) throws IOException {
        String data = System.getProperties()
                            .entrySet()
                            .stream()
                            .map(e->String.format("{\"%s\":\"%s\"}", e.getKey(), e.getValue()))
                            .collect(Collectors.joining(",\n"));

        send(t, 200, "[" + data + "]");
    }

    private static void suspend(HttpExchange t) throws IOException {
        Cluster cluster = CacheFactory.ensureCluster();
        cluster.suspendService("PartitionedCache");
        send(t, 200, "OK");
    }

    private static void resume(HttpExchange t) throws IOException {
        Cluster cluster = CacheFactory.ensureCluster();
        cluster.resumeService("PartitionedCache");
        send(t, 200, "OK");
    }

    private static void populate(HttpExchange t) throws IOException {
        populateCache(CacheFactory.getCache("cache-1"), 100);
        populateCache(CacheFactory.getCache("cache-2"), 100);
        send(t, 200, "OK");
    }

    private static void populateFlash(HttpExchange t) throws IOException {
        populateCache(CacheFactory.getCache("flash-1"), 1000);
        populateCache(CacheFactory.getCache("flash-2"), 1000);
        send(t, 200, "OK");
    }

    private static void populateRam(HttpExchange t) throws IOException {
        populateCache(CacheFactory.getCache("ram-1"), 1000);
        populateCache(CacheFactory.getCache("ram-2"), 1000);
        send(t, 200, "OK");
    }

    private static void populateFederation(HttpExchange t) throws IOException {
        populateCache(CacheFactory.getCache("federated-1"), 10000);
        populateCache(CacheFactory.getCache("federated-2"), 10000);
        populateCache(CacheFactory.getCache("federated-3"), 10000);
        send(t, 200, "OK");
    }

    private static void edition(HttpExchange t) throws IOException {
        send(t, 200, CacheFactory.getEdition());
    }

    private static void version(HttpExchange t) throws IOException {
        send(t, 200, CacheFactory.VERSION);
    }

    private static void isExecutorPresent(HttpExchange t) throws IOException {
        send(t, 200, Boolean.toString(canFindExecutor()));
    }

    private static void isHealthCheckPresent(HttpExchange t) throws IOException {
        send(t, 200, Boolean.toString(canFindHealthCheck()));
    }

    private static boolean canFindExecutor() {
        try {
            Class.forName("com.oracle.coherence.concurrent.executor.ClusteredExecutorInfo");
            return true;
        }
        catch (ClassNotFoundException e) {
            return false;
        }
    }

    private static boolean canFindHealthCheck() {
        try {
            Class.forName("com.tangosol.util.HealthCheck");
            return true;
        }
        catch (ClassNotFoundException e) {
            return false;
        }
    }

    /**
     * Registers Coherence*Web MBeans via reflection only so this compiles against CE.
     */
    private static void registerMBeans(HttpExchange t) throws IOException {
        try {
            Class<?> clazz          = Class.forName("com.oracle.coherence.cli.testing.ge.RegisterMockCWebMBean");
            Object   inst           = clazz.getDeclaredConstructor().newInstance();
            Method   registerMethod = clazz.getMethod("register", String.class);
            registerMethod.invoke(inst, "application1");
        }
        catch (Exception e) {
            send(t, 404, "Error");
        }
        send(t, 200, "OK");
    }

    private static void populateCache(NamedCache<Integer, String> cache, int count) {
        cache.clear();
        Map<Integer, String> map = new HashMap<>();

        for (int i = 0; i < count; i++) {
            map.put(i, "value-" + i);
            if (count % 1000 == 0) {
                cache.putAll(map);
                map.clear();
            }
        }
        if (!map.isEmpty()) {
            cache.putAll(map);
        }
    }
}
