/*
 * Copyright (c) 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */


package com.oracle.coherence.demo.frameworks.micronaut;

import io.micronaut.runtime.Micronaut;

/**
 * {@code Micronaut} entry point.
 */
public class Application
    {
    public static void main(String[] args)
        {
        Micronaut.run(Application.class, args);
        }
    }
