/*
 * Copyright (c) 2024, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.demo.frameworks.helidon;

import static jakarta.ws.rs.core.MediaType.APPLICATION_JSON;

import com.oracle.coherence.cdi.WhereFilter;
import com.oracle.coherence.cdi.events.*;
import com.oracle.coherence.common.base.Logger;
import com.tangosol.net.NamedMap;

import com.tangosol.util.MapEvent;
import jakarta.enterprise.context.ApplicationScoped;
import jakarta.enterprise.event.Observes;
import jakarta.inject.Inject;

import jakarta.ws.rs.Consumes;
import jakarta.ws.rs.DELETE;
import jakarta.ws.rs.GET;
import jakarta.ws.rs.POST;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.PathParam;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.Response;

@Path("/api/customers")
@ApplicationScoped
public class CustomerResource {

    @Inject
    private NamedMap<Integer, Customer> customers;

    @POST
    @Consumes(APPLICATION_JSON)
    public Response createCustomer(Customer customer) {
        customers.put(customer.getId(), customer);
        return Response.accepted().build();
    }

    @GET
    @Produces(APPLICATION_JSON)
    public Response getCustomers() {
        return Response.ok(customers.values()).build();
    }

    @GET
    @Path("{id}")
    @Produces(APPLICATION_JSON)
    public Response getTask(@PathParam("id") int id) {
        Customer customer = customers.get(id);

        return customer == null ? Response.status(Response.Status.NOT_FOUND).build() : Response.ok(customer).build();
    }

    @DELETE
    @Path("{id}")
    @Produces(APPLICATION_JSON)
    public Response deleteTask(@PathParam("id") int id) {
        Customer oldCustomer = customers.remove(id);
        return oldCustomer == null ? Response.status(Response.Status.NOT_FOUND).build() : Response.ok(oldCustomer).build();
    }

    // --- Register Map Listeners

    /**
     * Event fired on inserting of a {@link Customer}.
     * @param event event information
     */
    private void onCustomerInserted(@Observes @Inserted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Inserted: id=" + event.getKey() + ", value=" + event.getNewValue());
    }

    /**
     * Event fired on updating of a {@link Customer}.
     * @param event event information
     */
    private void onCustomerUpdated(@Observes @Updated @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Updated: id=" + event.getKey() + ", new value=" + event.getNewValue() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deletion a {@link Customer}.
     * @param event event information
     */
    private void onCustomerDeleted(@Observes @Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Deleted: id=" + event.getKey() + ", old value=" + event.getOldValue());
    }

    /**
     * Event fired on deleting of a {@link Customer} when they have a large balance > 5000.
     * @param event event information
     */
    @WhereFilter("balance > 5000.0d")
    private void onCustomerDeletedLargeBalance(@Observes @Deleted @MapName("customers") MapEvent<Integer, Customer> event) {
        Logger.info("Customer Deleted: (Large Balance) id=" + event.getKey() + ", old value=" + event.getOldValue());
    }
}
