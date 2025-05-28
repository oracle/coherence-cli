/*
 * Copyright (c) 2024, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.demo.frameworks.helidon;

import static jakarta.ws.rs.core.MediaType.APPLICATION_JSON;

import com.tangosol.net.NamedMap;

import jakarta.enterprise.context.ApplicationScoped;
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
        return Response.accepted(customer).build();
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
}
