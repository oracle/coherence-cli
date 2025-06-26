#
# Copyright (c) 2025, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at
# https://oss.oracle.com/licenses/upl.
#

from typing import List
import jsonpickle
import quart
from coherence import NamedMap, Session, Filters, Processors
from coherence.event import MapListener, MapEventFilter
from dataclasses import dataclass
from coherence.serialization import proxy
from quart import Quart, request, redirect


@dataclass
class Customer:
    id: int
    name: str
    balance: float


# ---- init ------------

# the Quart application.  Quart was chosen over Flask due to better
# handling of asyncio which is required to use the Coherence client
# library
app: Quart = Quart(__name__,
                   static_url_path='',
                   static_folder='./')


# the Session with the gRPC proxy
session: Session

customers: NamedMap[int, Customer]


@app.before_serving
async def init():

    # initialize the session using the default localhost:1408 or the value of COHERENCE_SERVER_ADDRESS
    global session
    session = await Session.create()

    global customers
    customers = await session.get_map('customers')

    print('adding listeners...')
    listener: MapListener[int, Customer] = MapListener()
    listener.on_updated(lambda e: handle_event(e))
    listener.on_inserted(lambda e: handle_event(e))
    listener.on_deleted(lambda e: handle_event(e))
    await customers.add_map_listener(listener)

    deleteListener: MapListener[int, Customer] = MapListener()
    deleteListener.on_deleted(lambda e: handle_delete_event(e))
    delete_filter = Filters.event(Filters.greater("balance", 5000), MapEventFilter.DELETED)
    await customers.add_map_listener(deleteListener, delete_filter)


# ----- routes --------------------------------------------------------------

# Get all customers
@app.route('/api/customers', methods=['GET'])
async def get_customers():
    customers_list: List[Customer] = []
    async for customer in await customers.values():
        customers_list.append(customer)

    return quart.Response(jsonpickle.encode(customers_list, unpicklable=False), mimetype="application/json")


# Create a person with JSON as body
@app.route('/api/customers', methods=['POST'])
async def create_customer():
    try:
        data = await request.get_json(force=True)
        name: str = data['name']
        id: int = int(data['id'])
        balance: float = float(data['balance'])
    except:
        return quart.Response(f"Invalid JSON", status=400)

    person: Customer = Customer(id, name, balance)
    await customers.put(person.id, person)

    return quart.Response(
        jsonpickle.encode(person, unpicklable=False),
        status=202,
        mimetype='application/json'
    )


# Get a single person
@app.route('/api/customers/<id>', methods=['GET'])
async def get_person(id: str):
    existing: Customer = await customers.get(int(id))
    if existing == None:
        return "", 404

    return jsonpickle.encode(existing, unpicklable=False), 200

# Delete a person
@app.route('/api/customers/<id>', methods=['DELETE'])
async def delete_person(id: str):
    """
    This route will delete the person with the given id.

    :param id: the id of the person to delete
    """
    existing: Customer = await customers.remove(int(id))
    return "", 404 if existing is None else 200

def handle_event(e) -> None:
    """
    Event handler to display the event details

    :return: None
    """
    key = e.key
    newValue = e.new
    oldValue = e.old

    print(
        f"Event {e.type} for key={key}, new={newValue}, old={oldValue}")


def handle_delete_event(e) -> None:
    """
    Event handler to display the event details

    :return: None
    """
    key = e.key
    oldValue = e.old

    print(
        f"Event delete large balance for key={key}, old={oldValue}")


# ----- main ----------------------------------------------------------------

if __name__ == '__main__':
    # run the application on port 8080
    app.run(host='0.0.0.0', port=8080)