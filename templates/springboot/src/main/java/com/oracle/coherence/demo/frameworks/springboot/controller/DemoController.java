/*
 * Copyright (c) 2024, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.demo.frameworks.springboot.controller;

import com.oracle.coherence.demo.frameworks.springboot.Customer;
import com.oracle.coherence.spring.configuration.annotation.CoherenceCache;

import com.tangosol.net.NamedCache;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

import java.util.Collection;

@RestController
@RequestMapping(path = "/api/customers")
public class DemoController {

    private final NamedCache<Integer, Customer> customers;

    public DemoController(@CoherenceCache NamedCache<Integer, Customer> customers) {
        this.customers = customers;
    }

    @GetMapping
    public Collection<Customer> getCustomers() {
        return customers.values();
    }

    @PostMapping
    public ResponseEntity<Void> createCustomer(@RequestBody Customer customer) {
        customers.put(customer.getId(), customer);
        return ResponseEntity.accepted().build();
    }

    @GetMapping("/{id}")
    public ResponseEntity<Customer> getCustomer(@PathVariable int id) {
        Customer customer = customers.get(id);
        return customer == null ? ResponseEntity.notFound().build() : ResponseEntity.ok(customer);
    }

    @DeleteMapping("/{id}")
    public void removeCustomer(@PathVariable int id) {
        customers.remove(id);
    }

}
