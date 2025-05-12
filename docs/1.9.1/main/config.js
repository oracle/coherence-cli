function createConfig() {
    return {
        home: "docs/about/overview",
        release: "1.9.1",
        releases: [
            "1.9.1"
        ],
        pathColors: {
            "*": "blue-grey"
        },
        theme: {
            primary: '#1976D2',
            secondary: '#424242',
            accent: '#82B1FF',
            error: '#FF5252',
            info: '#2196F3',
            success: '#4CAF50',
            warning: '#FFC107'
        },
        navTitle: 'Coherence CLI',
        navIcon: null,
        navLogo: '/images/logo.png'
    };
}

function createRoutes(){
    return [
        {
            path: '/docs/about/overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: 'Coherence CLI Overview',
                keywords: 'oracle coherence, coherence-cli, documentation',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-overview', 'docs/about/overview', {})
        },
        {
            path: '/docs/about/introduction',
            meta: {
                h1: 'Coherence CLI Introduction',
                title: 'Coherence CLI Introduction',
                h1Prefix: null,
                description: 'Coherence CLI Introduction',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-introduction', 'docs/about/introduction', {})
        },
        {
            path: '/docs/about/quickstart',
            meta: {
                h1: 'Quick Start',
                title: 'Quick Start',
                h1Prefix: null,
                description: 'Coherence CLI Quickstart',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, quickstart',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-quickstart', 'docs/about/quickstart', {})
        },
        {
            path: '/docs/installation/installation',
            meta: {
                h1: 'Coherence CLI Installation',
                title: 'Coherence CLI Installation',
                h1Prefix: null,
                description: 'Coherence CLI - Coherence CLI Installation',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Coherence CLI Installation, Mac, Linux, Windows',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-installation-installation', 'docs/installation/installation', {})
        },
        {
            path: '/docs/config/overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: 'Coherence CLI - Configuration Overview',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Configuration Overview',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-overview', 'docs/config/overview', {})
        },
        {
            path: '/docs/config/global_flags',
            meta: {
                h1: 'Global Flags',
                title: 'Global Flags',
                h1Prefix: null,
                description: 'Coherence CLI - Global Flags',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Global Flags',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-global_flags', 'docs/config/global_flags', {})
        },
        {
            path: '/docs/config/bytes_display_format',
            meta: {
                h1: 'Bytes Display Format',
                title: 'Bytes Display Format',
                h1Prefix: null,
                description: 'Coherence CLI - Bytes Display Format',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Bytes Display Format',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-bytes_display_format', 'docs/config/bytes_display_format', {})
        },
        {
            path: '/docs/config/command_completion',
            meta: {
                h1: 'Command Completion',
                title: 'Command Completion',
                h1Prefix: null,
                description: 'Coherence CLI - Command Completion',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Command Completion',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-command_completion', 'docs/config/command_completion', {})
        },
        {
            path: '/docs/config/sorting_table_output',
            meta: {
                h1: 'Sorting Table Output',
                title: 'Sorting Table Output',
                h1Prefix: null,
                description: 'Coherence CLI - Sorting Table Output',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Sorting Table Output',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-sorting_table_output', 'docs/config/sorting_table_output', {})
        },
        {
            path: '/docs/config/get_config',
            meta: {
                h1: 'Get Config',
                title: 'Get Config',
                h1Prefix: null,
                description: 'Coherence CLI - Get Config',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Get Config',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-get_config', 'docs/config/get_config', {})
        },
        {
            path: '/docs/config/changing_config_locations',
            meta: {
                h1: 'Changing Config Locations',
                title: 'Changing Config Locations',
                h1Prefix: null,
                description: 'Coherence CLI - Changing Config Locations',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Changing Config Locations',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-changing_config_locations', 'docs/config/changing_config_locations', {})
        },
        {
            path: '/docs/config/using_proxy_servers',
            meta: {
                h1: 'Using Proxy Servers',
                title: 'Using Proxy Servers',
                h1Prefix: null,
                description: 'Coherence CLI - Using Proxy Servers',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Using Proxy Servers',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-using_proxy_servers', 'docs/config/using_proxy_servers', {})
        },
        {
            path: '/docs/reference/overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: 'Coherence CLI - Commands Overview',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, commands overview',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-overview', 'docs/reference/overview', {})
        },
        {
            path: '/docs/reference/clusters',
            meta: {
                h1: 'Clusters',
                title: 'Clusters',
                h1Prefix: null,
                description: 'Coherence CLI - Cluster Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, cluster commands, create, start, stop, scale',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-clusters', 'docs/reference/clusters', {})
        },
        {
            path: '/docs/reference/contexts',
            meta: {
                h1: 'Contexts',
                title: 'Contexts',
                h1Prefix: null,
                description: 'Coherence CLI - Using Context Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Using Context',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-contexts', 'docs/reference/contexts', {})
        },
        {
            path: '/docs/reference/members',
            meta: {
                h1: 'Members',
                title: 'Members',
                h1Prefix: null,
                description: 'Coherence CLI - Members Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Members Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-members', 'docs/reference/members', {})
        },
        {
            path: '/docs/reference/machines',
            meta: {
                h1: 'Machines',
                title: 'Machines',
                h1Prefix: null,
                description: 'Coherence CLI - Machines Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Machines Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-machines', 'docs/reference/machines', {})
        },
        {
            path: '/docs/reference/services',
            meta: {
                h1: 'Services',
                title: 'Services',
                h1Prefix: null,
                description: 'Coherence CLI - Services Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, services commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-services', 'docs/reference/services', {})
        },
        {
            path: '/docs/reference/caches',
            meta: {
                h1: 'Caches',
                title: 'Caches',
                h1Prefix: null,
                description: 'Coherence CLI - Cache Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Cache Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-caches', 'docs/reference/caches', {})
        },
        {
            path: '/docs/reference/view_caches',
            meta: {
                h1: 'View Caches',
                title: 'View Caches',
                h1Prefix: null,
                description: 'Coherence CLI - View Cache Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, view cache commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-view_caches', 'docs/reference/view_caches', {})
        },
        {
            path: '/docs/reference/topics',
            meta: {
                h1: 'Topics',
                title: 'Topics',
                h1Prefix: null,
                description: 'Coherence CLI - Topics Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, topics commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-topics', 'docs/reference/topics', {})
        },
        {
            path: '/docs/reference/persistence',
            meta: {
                h1: 'Persistence',
                title: 'Persistence',
                h1Prefix: null,
                description: 'Coherence CLI - Persistence Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Persistence commands, snapshot, archive snapshot',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-persistence', 'docs/reference/persistence', {})
        },
        {
            path: '/docs/reference/federation',
            meta: {
                h1: 'Federation',
                title: 'Federation',
                h1Prefix: null,
                description: 'Coherence CLI - Federation Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Federation Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-federation', 'docs/reference/federation', {})
        },
        {
            path: '/docs/reference/nslookup',
            meta: {
                h1: 'NS Lookup',
                title: 'NS Lookup',
                h1Prefix: null,
                description: 'Coherence CLI - NSLookup Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, NSLookup commands, name service',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-nslookup', 'docs/reference/nslookup', {})
        },
        {
            path: '/docs/reference/proxies',
            meta: {
                h1: 'Proxy Servers',
                title: 'Proxy Servers',
                h1Prefix: null,
                description: 'Coherence CLI - Proxy Server Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, proxy servers commands, Cohernce Extend',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-proxies', 'docs/reference/proxies', {})
        },
        {
            path: '/docs/reference/http_servers',
            meta: {
                h1: 'Http Servers',
                title: 'Http Servers',
                h1Prefix: null,
                description: 'Coherence CLI - Http Servers Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Http Servers Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-http_servers', 'docs/reference/http_servers', {})
        },
        {
            path: '/docs/reference/elastic_data',
            meta: {
                h1: 'Elastic Data',
                title: 'Elastic Data',
                h1Prefix: null,
                description: 'Coherence CLI - Elastic Data Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Elastic Data Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-elastic_data', 'docs/reference/elastic_data', {})
        },
        {
            path: '/docs/reference/http_sessions',
            meta: {
                h1: 'Http Sessions',
                title: 'Http Sessions',
                h1Prefix: null,
                description: 'Coherence CLI - Http Sessions Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Http Sessions Commands, Coherence Web',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-http_sessions', 'docs/reference/http_sessions', {})
        },
        {
            path: '/docs/reference/executors',
            meta: {
                h1: 'Executors',
                title: 'Executors',
                h1Prefix: null,
                description: 'Coherence CLI - Executors Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Executors Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-executors', 'docs/reference/executors', {})
        },
        {
            path: '/docs/reference/reporters',
            meta: {
                h1: 'Reporters',
                title: 'Reporters',
                h1Prefix: null,
                description: 'Coherence CLI - Reporters Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, reporter commands,',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-reporters', 'docs/reference/reporters', {})
        },
        {
            path: '/docs/reference/diagnostics',
            meta: {
                h1: 'Diagnostics',
                title: 'Diagnostics',
                h1Prefix: null,
                description: 'Coherence CLI - Diagnostics Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, diagnostic commands, jfr, heap dump, tracing',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-diagnostics', 'docs/reference/diagnostics', {})
        },
        {
            path: '/docs/reference/health',
            meta: {
                h1: 'Health',
                title: 'Health',
                h1Prefix: null,
                description: 'Coherence CLI - Health Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Health Commands, monitor',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-health', 'docs/reference/health', {})
        },
        {
            path: '/docs/reference/reset',
            meta: {
                h1: 'Resetting Statistics',
                title: 'Resetting Statistics',
                h1Prefix: null,
                description: 'Coherence CLI - Resetting Statistics',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Resetting Statistics, cache, executor, federation, service, member',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-reset', 'docs/reference/reset', {})
        },
        {
            path: '/docs/reference/misc',
            meta: {
                h1: 'Miscellaneous',
                title: 'Miscellaneous',
                h1Prefix: null,
                description: 'Coherence CLI - Miscellaneous Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Miscellaneous Commands, timeout, debug, color',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-misc', 'docs/reference/misc', {})
        },
        {
            path: '/docs/reference/create_clusters',
            meta: {
                h1: 'Creating Clusters',
                title: 'Creating Clusters',
                h1Prefix: null,
                description: 'Coherence CLI - Creating Clusters',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Creating Clusters, development, experimental',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-create_clusters', 'docs/reference/create_clusters', {})
        },
        {
            path: '/docs/reference/monitor_clusters',
            meta: {
                h1: 'Monitor Clusters',
                title: 'Monitor Clusters',
                h1Prefix: null,
                description: 'Coherence CLI - Monitor Clusters Commands',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Monitor Clusters Commands',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-monitor_clusters', 'docs/reference/monitor_clusters', {})
        },
        {
            path: '/docs/reference/create_starter',
            meta: {
                h1: 'Starter Projects',
                title: 'Starter Projects',
                h1Prefix: null,
                description: 'Coherence CLI - Create Starter Projects',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Starter Projects, Spring, Helidon, Micronaut, SpringBoot',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-create_starter', 'docs/reference/create_starter', {})
        },
        {
            path: '/docs/security/overview',
            meta: {
                h1: 'Securing CLI Access',
                title: 'Securing CLI Access',
                h1Prefix: null,
                description: 'Coherence CLI - Securing CLI Access',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Securing CLI Access',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-security-overview', 'docs/security/overview', {})
        },
        {
            path: '/docs/troubleshooting/trouble-shooting',
            meta: {
                h1: 'Troubleshooting Guide',
                title: 'Troubleshooting Guide',
                h1Prefix: null,
                description: 'Coherence CLI - Troubleshooting Guide',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Troubleshooting Guide',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-troubleshooting-trouble-shooting', 'docs/troubleshooting/trouble-shooting', {})
        },
        {
            path: '/docs/examples/overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: 'Coherence CLI - Examples Overview',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, CLI examples Overview',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-overview', 'docs/examples/overview', {})
        },
        {
            path: '/docs/examples/rolling_restarts',
            meta: {
                h1: 'Rolling Restarts',
                title: 'Rolling Restarts',
                h1Prefix: null,
                description: 'Coherence CLI - Rolling Restarts',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, rolling, restarts',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-rolling_restarts', 'docs/examples/rolling_restarts', {})
        },
        {
            path: '/docs/examples/jsonpath',
            meta: {
                h1: 'Using JSONPath',
                title: 'Using JSONPath',
                h1Prefix: null,
                description: 'Coherence CLI - Using JSONPath',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, jsonpath,',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-jsonpath', 'docs/examples/jsonpath', {})
        },
        {
            path: '/docs/examples/set_cache_attrs',
            meta: {
                h1: 'Setting Cache Attributes',
                title: 'Setting Cache Attributes',
                h1Prefix: null,
                description: 'Coherence CLI - Setting Cache Attributes',
                keywords: 'oracle coherence, coherence-cli, documentation, management, cli, Setting Cache Attributes',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-set_cache_attrs', 'docs/examples/set_cache_attrs', {})
        },
        {
            path: '/', redirect: 'docs/about/overview'
        },
        {
            path: '*', redirect: '/'
        }
    ];
}
function createNav(){
    return [
        {
            type: 'page',
            title: 'Core documentation',
            to: '/docs/about/overview',
            action: null
        },
        {
            type: 'menu',
            title: 'About',
            group: '/docs/about',
            items: [
                {
                    type: 'page',
                    title: 'Overview',
                    to: '/docs/about/overview',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Coherence CLI Introduction',
                    to: '/docs/about/introduction',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Quick Start',
                    to: '/docs/about/quickstart',
                    action: null
                }
            ],
            action: 'assistant'
        },
        {
            type: 'page',
            title: 'Installation',
            to: '/docs/installation/installation',
            action: 'fa-save'
        },
        {
            type: 'menu',
            title: 'Configuration',
            group: '/docs/config',
            items: [
                {
                    type: 'page',
                    title: 'Overview',
                    to: '/docs/config/overview',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Global Flags',
                    to: '/docs/config/global_flags',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Bytes Display Format',
                    to: '/docs/config/bytes_display_format',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Command Completion',
                    to: '/docs/config/command_completion',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Sorting Table Output',
                    to: '/docs/config/sorting_table_output',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Get Config',
                    to: '/docs/config/get_config',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Changing Config Locations',
                    to: '/docs/config/changing_config_locations',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Using Proxy Servers',
                    to: '/docs/config/using_proxy_servers',
                    action: null
                }
            ],
            action: 'fa-cogs'
        },
        {
            type: 'menu',
            title: 'Command Reference',
            group: '/docs/reference',
            items: [
                {
                    type: 'page',
                    title: 'Overview',
                    to: '/docs/reference/overview',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Clusters',
                    to: '/docs/reference/clusters',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Contexts',
                    to: '/docs/reference/contexts',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Members',
                    to: '/docs/reference/members',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Machines',
                    to: '/docs/reference/machines',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Services',
                    to: '/docs/reference/services',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Caches',
                    to: '/docs/reference/caches',
                    action: null
                },
                {
                    type: 'page',
                    title: 'View Caches',
                    to: '/docs/reference/view_caches',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Topics',
                    to: '/docs/reference/topics',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Persistence',
                    to: '/docs/reference/persistence',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Federation',
                    to: '/docs/reference/federation',
                    action: null
                },
                {
                    type: 'page',
                    title: 'NS Lookup',
                    to: '/docs/reference/nslookup',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Proxy Servers',
                    to: '/docs/reference/proxies',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Http Servers',
                    to: '/docs/reference/http_servers',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Elastic Data',
                    to: '/docs/reference/elastic_data',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Http Sessions',
                    to: '/docs/reference/http_sessions',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Executors',
                    to: '/docs/reference/executors',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Reporters',
                    to: '/docs/reference/reporters',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Diagnostics',
                    to: '/docs/reference/diagnostics',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Health',
                    to: '/docs/reference/health',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Resetting Statistics',
                    to: '/docs/reference/reset',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Miscellaneous',
                    to: '/docs/reference/misc',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Creating Clusters',
                    to: '/docs/reference/create_clusters',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Monitor Clusters',
                    to: '/docs/reference/monitor_clusters',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Starter Projects',
                    to: '/docs/reference/create_starter',
                    action: null
                }
            ],
            action: 'find_in_page'
        },
        {
            type: 'page',
            title: 'Security',
            to: '/docs/security/overview',
            action: 'lock'
        },
        {
            type: 'page',
            title: 'Troubleshooting',
            to: '/docs/troubleshooting/trouble-shooting',
            action: 'fa-question-circle'
        },
        {
            type: 'menu',
            title: 'Examples',
            group: '/docs/examples',
            items: [
                {
                    type: 'page',
                    title: 'Overview',
                    to: '/docs/examples/overview',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Rolling Restarts',
                    to: '/docs/examples/rolling_restarts',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Using JSONPath',
                    to: '/docs/examples/jsonpath',
                    action: null
                },
                {
                    type: 'page',
                    title: 'Setting Cache Attributes',
                    to: '/docs/examples/set_cache_attrs',
                    action: null
                }
            ],
            action: 'explore'
        },
        {
            type: 'header',
            title: 'Additional resources',
            action: null
        },
        {
            type: 'link',
            title: 'Slack',
            href: 'https://join.slack.com/t/oraclecoherence/shared_invite/enQtNzcxNTQwMTAzNjE4LTJkZWI5ZDkzNGEzOTllZDgwZDU3NGM2YjY5YWYwMzM3ODdkNTU2NmNmNDFhOWIxMDZlNjg2MzE3NmMxZWMxMWE',
            target: '_blank',
            action: 'fa-slack'
        },
        {
            type: 'link',
            title: 'Coherence Community',
            href: 'https://coherence.community',
            target: '_blank',
            action: 'people'
        },
        {
            type: 'link',
            title: 'GitHub',
            href: 'https://github.com/oracle/coherence-cli',
            target: '_blank',
            action: 'fa-github-square'
        }
    ];
}
