/*
 * Copyright (c) 2021, 2025 Oracle and/or its affiliates.
 * Licensed under the Universal Permissive License v 1.0 as shown at
 * https://oss.oracle.com/licenses/upl.
 */

package com.oracle.coherence.cli.testing.ge;

import com.tangosol.util.Base;
import java.util.Date;

import com.tangosol.coherence.servlet.management.HttpSessionManagerMBean;

import com.tangosol.net.CacheFactory;

import com.tangosol.net.management.AnnotatedStandardMBean;
import com.tangosol.net.management.Registry;

import javax.management.NotCompliantMBeanException;

/**
 * Register a mock Coherence*Web MBean for testing.
 *
 * @author tam 2021.10.25
 */
public class RegisterMockCWebMBean {

    public void register(String applicationId) {
        Registry registry = CacheFactory.getCluster().getManagement();
        if (registry != null) {
            String                 sName = registry.ensureGlobalName(HttpSessionManagerMBean.OBJECT_TYPE + ",appId=" + applicationId);
            AnnotatedStandardMBean mbean;
            try {
                mbean = new AnnotatedStandardMBean(new MockHttpSessionManagerMBeanImpl(), HttpSessionManagerMBean.class, true);
            }
            catch (NotCompliantMBeanException e) {
                throw Base.ensureRuntimeException(e);
            }
            registry.register(sName, mbean);
        }
    }

    /**
     * The mock {@link HttpSessionManagerMBean} implementation for testing purposes.
     */
    public static class MockHttpSessionManagerMBeanImpl
            extends Base
            implements HttpSessionManagerMBean {

        @Override
        public long getAverageReapDuration() {
            return 30;
        }

        @Override
        public long getAverageReapedSessions() {
            return 0;
        }

        @Override
        public String getCollectionClassName() {
            return null;
        }

        @Override
        public String getFactoryClassName() {
            return null;
        }

        @Override
        public long getLastReapDuration() {
            return 0;
        }

        @Override
        public Date getLastReapCycle() {
            return null;
        }

        @Override
        public String getLocalAttributeCacheName() {
            return null;
        }

        @Override
        public int getLocalAttributeCount() {
            return 0;
        }

        @Override
        public String getLocalSessionCacheName() {
            return null;
        }

        @Override
        public int getLocalSessionCount() {
            return 0;
        }

        @Override
        public long getMaxReapedSessions() {
            return 0;
        }

        @Override
        public long getMaxReapDuration() {
            return 0;
        }

        @Override
        public long getAverageReapQueueWaitDuration() {
            return 0;
        }

        @Override
        public long getLastReapQueueWaitDuration() {
            return 0;
        }

        @Override
        public long getMaxReapQueueWaitDuration() {
            return 0;
        }

        @Override
        public Date getNextReapCycle() {
            return null;
        }

        @Override
        public int getOverflowAverageSize() {
            return 0;
        }

        @Override
        public String getOverflowCacheName() {
            return null;
        }

        @Override
        public int getOverflowMaxSize() {
            return 0;
        }

        @Override
        public int getOverflowThreshold() {
            return 0;
        }

        @Override
        public int getOverflowUpdates() {
            return 0;
        }

        @Override
        public long getReapedSessions() {
            return 0;
        }

        @Override
        public long getReapedSessionsTotal() {
            return 0;
        }

        @Override
        public int getSessionAverageLifetime() {
            return 0;
        }

        @Override
        public int getSessionAverageSize() {
            return 0;
        }

        @Override
        public String getSessionCacheName() {
            return "testcache";
        }

        @Override
        public int getSessionIdLength() {
            return 0;
        }

        @Override
        public int getSessionMaxSize() {
            return 0;
        }

        @Override
        public int getSessionMinSize() {
            return 0;
        }

        @Override
        public int getSessionStickyCount() {
            return 0;
        }

        @Override
        public int getSessionTimeout() {
            return 0;
        }

        @Override
        public int getSessionUpdates() {
            return 0;
        }

        @Override
        public String getServletContextCacheName() {
            return null;
        }

        @Override
        public String getServletContextName() {
            return null;
        }

        @Override
        public void resetStatistics() {
        }

        @Override
        public void clearStoredConfiguration() {
        }

        // required for 14.1.2.0.4
        public void setSessionDebugLogging(boolean fDebug) {
        }

        public boolean isSessionDebugLogging() {
            return false;
        }
    }
}
