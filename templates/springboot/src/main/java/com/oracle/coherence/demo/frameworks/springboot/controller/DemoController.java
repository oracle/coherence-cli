/*
 * Copyright (c) 2024, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.demo.frameworks.springboot.controller;

import com.oracle.coherence.common.base.Logger;

import com.oracle.coherence.demo.frameworks.springboot.Customer;

import com.oracle.coherence.spring.annotation.WhereFilter;
import com.oracle.coherence.spring.annotation.event.Deleted;
import com.oracle.coherence.spring.annotation.event.Inserted;
import com.oracle.coherence.spring.annotation.event.MapName;
import com.oracle.coherence.spring.annotation.event.Updated;
import com.oracle.coherence.spring.configuration.annotation.CoherenceCache;

import com.oracle.coherence.spring.event.CoherenceEventListener;

import com.tangosol.net.NamedCache;

import com.tangosol.util.MapEvent;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
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

    // --- Register Map Listeners

    /**
     * Event fired on inserting of a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    private void onCustomerInserted(@Inserted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Inserted: id=" + event.getKey() + ", value=" + event.getNewValue());
    }

    /**
     * Event fired on updating of a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    private void onCustomerUpdated(@Updated @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Updated: id=" + event.getKey() + ", new value=" + event.getNewValue() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deletion a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    private void onCustomerDeleted(@Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Deleted: id=" + event.getKey() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deleting of a {@link Customer} when they have a large balance > 5000.
     * @param event event information
     */
    @CoherenceEventListener
    @WhereFilter("balance > 5000.0d")
    private void onCustomerDeletedLargeBalance(@Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Deleted: (Large Balance) id=" + event.getKey() + ", old value=" + event.getOldValue());
    }
}
