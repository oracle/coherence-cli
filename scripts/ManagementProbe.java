/*
 * Copyright (c) 2026 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

import java.lang.management.ManagementFactory;
import java.rmi.registry.LocateRegistry;
import java.util.Arrays;
import javax.management.ObjectName;
import javax.management.remote.JMXConnector;
import javax.management.remote.JMXConnectorFactory;
import javax.management.remote.JMXServiceURL;

/** JDK-only registry/RMI connectivity and harmless Runtime MBean identity check. */
public class ManagementProbe {
    public static void main(String[] args) throws Exception {
        String host = args[0];
        int port = Integer.parseInt(args[1]);
        String expectedPid = args[2];
        System.setProperty("sun.rmi.transport.tcp.responseTimeout", "3000");
        String[] bindings = LocateRegistry.getRegistry(host, port).list();
        if (!Arrays.asList(bindings).contains("jmxrmi")) {
            throw new IllegalStateException("JMX registry binding missing");
        }
        System.out.println("JMX registry binding: " + LocateRegistry.getRegistry(host, port).lookup("jmxrmi"));
        JMXServiceURL url = new JMXServiceURL("service:jmx:rmi:///jndi/rmi://" + host + ":" + port + "/jmxrmi");
        try (JMXConnector connector = JMXConnectorFactory.connect(url)) {
            String name = (String) connector.getMBeanServerConnection().getAttribute(
                    new ObjectName(ManagementFactory.RUNTIME_MXBEAN_NAME), "Name");
            if (!name.startsWith(expectedPid + "@")) {
                throw new IllegalStateException("Unexpected JVM identity: " + name);
            }
            System.out.println("JMX registry and RMI connected: " + host + ":" + port + " runtime=" + name);
        }
    }
}
