/*
 * Copyright (c) 2022 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.cli.testing.topics;


import com.tangosol.net.Session;
import com.tangosol.net.topic.Publisher;
import com.tangosol.net.topic.Subscriber;
import com.tangosol.util.Base;
import java.util.Random;
import java.util.UUID;
import java.util.concurrent.CancellationException;

import com.tangosol.net.CacheFactory;

import static com.tangosol.net.topic.Subscriber.Name.inGroup;
import static com.tangosol.util.Base.log;

/**
 * Starts topics processes.
 *
 * @author tam 2022.11.31
 *
 */
public class RunTopics {

    private static Publisher<String>  publisherPublic;
    private static Subscriber<String> subscriberPublic;
    private static Publisher<String>  publisherPrivate;
    private static Subscriber<String> subscriberPrivate;

    private static Thread thread = null;
    public RunTopics() {
    }

    public void startTopics() {
        System.out.println("Start topics");

        thread = new Thread(new Runner());
        thread.setDaemon(false);
        thread.start();
    }

    public void stopTopics() {
        System.out.println("Stop topics");

        thread.stop();
    }

    public static Void receive(Subscriber.Element<String> element, Throwable throwable, Subscriber<String> subscriber) {
        if (throwable != null) {
            if (throwable instanceof CancellationException) {
                // exiting process, ignore.
            }
        }
        else {
            System.out.println("received: " + element.getValue());
            element.commit();
            subscriber.receive().handle((v, err)->receive(v, err, subscriber));
        }
        return null;
    }

    public static class Runner
            implements Runnable {

        public Runner() {
            Session session = Session.create();
            publisherPublic = session.createPublisher("public-messages");
            subscriberPublic = session.createSubscriber("public-messages");
            // create a subscriber to receive private messages

            subscriberPrivate = session.createSubscriber("private-messages", inGroup("1"));

            subscriberPublic.receive().handle((v, err)->receive(v, err, subscriberPublic));
            subscriberPrivate.receive().handle((v, err)->receive(v, err, subscriberPrivate));

            publisherPrivate = session.createPublisher("private-messages");
        }

        @Override
        public void run() {
            Random random = new Random();

            while (true) {

                publisherPublic.publish(UUID.randomUUID().toString()).join();
                System.out.println("Publish");
                Base.sleep((long) random.nextInt(1000));
            }
        }
    }
}
