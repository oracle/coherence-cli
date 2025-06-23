/*
 * Copyright (c) 2025, Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/oracle/coherence-go-client/v2/coherence"
	"github.com/oracle/coherence-go-client/v2/coherence/extractors"
	"github.com/oracle/coherence-go-client/v2/coherence/filters"
	"log"
	"net/http"
	"strconv"
)

// Customer defines a customer
type Customer struct {
	ID      int     `json:"id,omitempty"`
	Name    string  `json:"name,omitempty"`
	Balance float32 `json:"balance,omitempty"`
}

var (
	customers coherence.NamedCache[int, Customer]
	ctx       = context.Background()
)

func main() {
	var (
		ctx = context.Background()
	)

	// create a new Session to the default gRPC port of 1408 using plain text
	session, err := coherence.NewSession(ctx, coherence.WithPlainText())
	if err != nil {
		panic(err)
	}
	defer session.Close()

	customers, err = coherence.GetNamedCache[int, Customer](session, "customers")
	if err != nil {
		log.Println("unable to create cache 'customers'", err)
		return
	}

	fmt.Println("Adding listeners...")

	// add a listener to listen for all events
	listener := coherence.NewMapListener[int, Customer]().OnAny(func(e coherence.MapEvent[int, Customer]) {
		var (
			newValue *Customer
			oldValue *Customer
		)
		key, err1 := e.Key()
		if err1 != nil {
			panic("unable to deserialize key")
		}

		if e.Type() == coherence.EntryInserted || e.Type() == coherence.EntryUpdated {
			newValue, err1 = e.NewValue()
			if err1 != nil {
				panic("unable to deserialize new value")
			}
		}
		if e.Type() == coherence.EntryDeleted || e.Type() == coherence.EntryUpdated {
			oldValue, err1 = e.OldValue()
			if err1 != nil {
				panic("unable to deserialize old value")
			}
		}

		fmt.Printf("**EVENT=%v: key=%v, oldValue=%v, newValue=%v\n", e.Type(), *key, oldValue, newValue)
	})

	if customers.AddListener(ctx, listener) != nil {
		log.Fatalf("unable to add listener %v", err)
	}

	defer func() {
		_ = customers.RemoveListener(ctx, listener)
	}()

	// add a custom listener for large balance deletion
	listenerDelete := coherence.NewMapListener[int, Customer]().OnDeleted(func(e coherence.MapEvent[int, Customer]) {
		key, err1 := e.Key()
		if err1 != nil {
			panic("unable to deserialize key")
		}
		oldValue, err1 := e.OldValue()
		if err1 != nil {
			panic("unable to deserialize old value")
		}

		fmt.Printf("**EVENT=%v: Large Balance key=%v, oldValue=%v, \n", e.Type(), *key, oldValue)
	})

	filter := filters.GreaterEqual[float32](extractors.Extract[float32]("balance"), 5000)

	if customers.AddFilterListener(ctx, listenerDelete, filter) != nil {
		log.Fatalf("unable to add listener %v", err)
	}

	defer func() {
		_ = customers.RemoveListener(ctx, listenerDelete)
	}()

	http.HandleFunc("/api/customers", customerHandler)
	http.HandleFunc("/api/customers/", customerByIDHandler)

	fmt.Println("Server running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func customerHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var customer Customer
		if err := json.NewDecoder(r.Body).Decode(&customer); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		_, err := customers.Put(ctx, customer.ID, customer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.WriteHeader(http.StatusAccepted)

	case http.MethodGet:
		var customerList = make([]Customer, 0)

		for ch := range customers.Values(ctx) {
			if ch.Err != nil {
				http.Error(w, ch.Err.Error(), http.StatusInternalServerError)
				return
			}
			customerList = append(customerList, ch.Value)
		}
		_ = json.NewEncoder(w).Encode(customers)
		return
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func customerByIDHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/customers/"):]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		p, err1 := customers.Get(ctx, id)

		if err1 != nil {
			http.Error(w, err1.Error(), http.StatusInternalServerError)
		}
		if p == nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		err1 = json.NewEncoder(w).Encode(&p)
		if err1 != nil {
			http.Error(w, err1.Error(), http.StatusInternalServerError)
			return
		}

	case http.MethodDelete:
		old, err2 := customers.Remove(ctx, id)
		if err2 != nil {
			http.Error(w, err2.Error(), http.StatusInternalServerError)
			return
		}
		if old == nil {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
