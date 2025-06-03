/*
 * Copyright (c) 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */


package com.oracle.coherence.demo.frameworks.micronaut.rest;

import java.util.Collection;

import com.oracle.coherence.common.base.Logger;

import com.oracle.coherence.demo.frameworks.micronaut.Customer;

import com.tangosol.net.NamedCache;

import com.tangosol.util.MapEvent;

import io.micronaut.coherence.annotation.*;

import io.micronaut.http.HttpResponse;
import io.micronaut.http.MediaType;

import io.micronaut.http.annotation.Body;
import io.micronaut.http.annotation.Controller;
import io.micronaut.http.annotation.Delete;
import io.micronaut.http.annotation.Get;
import io.micronaut.http.annotation.PathVariable;
import io.micronaut.http.annotation.Post;

import jakarta.inject.Inject;
import jakarta.inject.Singleton;

@Controller("/api/customers")
@Singleton
public class ApplicationController {
    @Inject
    private NamedCache<Integer, Customer> customers;

    @Get(produces = MediaType.APPLICATION_JSON)
    public Collection<Customer> getCustomers() {
        return customers.values();
    }
    
    @Post(consumes = MediaType.APPLICATION_JSON, produces = MediaType.APPLICATION_JSON)
    public HttpResponse<Customer> createCustomer(@Body Customer customer) {
        customers.put(customer.getId(), customer);
        return HttpResponse.accepted();
    }
    
    @Get("/{id}")
    public HttpResponse<Customer> getCustomer(@PathVariable("id") int id) {
        Customer customer = customers.get(id);

        return customer == null ? HttpResponse.notFound() : HttpResponse.ok(customer);
    }

    @Delete("/{id}")
    public HttpResponse<Customer>  deleteCustomer(@PathVariable("id") int id) {
        Customer oldCustomer = customers.remove(id);
        return oldCustomer == null ? HttpResponse.notFound() : HttpResponse.accepted();
    }

    // --- Register Map Listeners

    /**
     * Event fired on inserting of a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    public void onCustomerInserted(@Inserted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Inserted: id=" + event.getKey() + ", value=" + event.getNewValue());
    }

    /**
     * Event fired on updating of a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    public void onCustomerUpdated(@Updated @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Updated: id=" + event.getKey() + ", new value=" + event.getNewValue() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deletion a {@link Customer}.
     * @param event event information
     */
    @CoherenceEventListener
    public void onCustomerDeleted(@Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Deleted: id=" + event.getKey() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deleting of a {@link Customer} when they have a large balance > 5000.
     * @param event event information
     */
    @CoherenceEventListener
    @WhereFilter("balance > 5000.0d")
    public void onCustomerDeletedLargeBalance(@Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Updated: (Large Balance) id=" + event.getKey() + ", old value=" + event.getOldValue());
    }
}
