/*
 * Copyright (c) 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

const coh = require('@oracle/coherence')

const Session = coh.Session
const MapListener = coh.event.MapListener
const MapEventType = coh.event.MapEventType
const Filters = coh.Filters

const express = require('express');
const port = process.env.PORT || 8080
const api = express();

api.use(express.json()); // to parse JSON request bodies

// setup session to Coherence
const session = new Session()
const customers = session.getCache('customers')

async function addListeners() {
    console.log("adding listeners")

    const handler = (event) => {
        let oldValue = event.oldValue
        let newValue = event.newValue
        let oldDescription = oldValue !== null ? "name=" + oldValue.name + ", balance=$" + oldValue.balance : "N/A"
        let newDescription = newValue !== null ? "name=" + newValue.name + ", balance=$" + newValue.balance : "N/A"

        console.log(event.description + ": " + event.key + ", new=" + newDescription + ", old=" + oldDescription);
    }
    const listener = new MapListener()
        .on(MapEventType.INSERT, handler)
        .on(MapEventType.DELETE, handler)
        .on(MapEventType.UPDATE, handler)

    const deleteHandler = (event) => {
        let oldValue = event.oldValue
        console.log("delete (Large balance): key=" + event.key + ", old=" + oldValue.name + ", balance=$" + oldValue.balance)
    }
    const listenerDelete = new MapListener().on(MapEventType.DELETE, deleteHandler)

    await customers.addMapListener(listener)

    const eventFilter = Filters.event(Filters.greater("balance", 5000))
    await customers.addMapListener(listenerDelete, eventFilter)
}

// ----- REST API -----------------------------------------------------------

/**
 * Returns all customers.
 */
api.get('/api/customers', (req, res, next) => {
    const toSend = []
    customers.values()
        .then(async values => {
            // copy values to array to be sent via express
            for await (let value of values) {
                toSend.push(value)
            }
            res.send(toSend)
        })
        .catch(err => next(err))
})


/**
 * Create a customer.
 */
api.post('/api/customers', (req, res, next) => {
    const id = Number(req.body.id);
    const customer = {
        id: req.body.id,
        name: req.body.name,
        balance: req.body.balance
    }

    customers.set(id, customer)
        .then(() => {
            res.sendStatus(202);
        })
        .catch(err => next(err))
})


/**
 * Get a single customer.
 */
api.get('/api/customers/:id', (req, res, next) => {
    const id = Number(req.params.id);
    customers.get(id)
        .then(customer => {
            if (customer) {
                res.status(200).json(customer);
            } else {
                res.sendStatus(404);
            }
        })
        .catch(err => next(err));
});

/**
 * Delete a customer.
 */
api.delete('/api/customers/:id', (req, res, next) => {
    const id = Number(req.params.id);
    customers.delete(id)
        .then(oldValue => {
            res.sendStatus(oldValue ? 200 : 404)
        })
        .catch(err => next(err))
})

addListeners().then(s => console.log("Listeners added"))

api.listen(port, () => console.log(`Listening on port ${port}`))
