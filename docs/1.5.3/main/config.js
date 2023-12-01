function createConfig() {
    return {
        home: "docs/about/01_overview",
        release: "1.5.3",
        releases: [
            "1.5.3"
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
        navLogo: 'images/logo.png'
    };
}

function createRoutes(){
    return [
        {
            path: '/docs/about/01_overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: 'Coherence CLI documentation',
                keywords: 'oracle coherence, coherence-cli, documentation',
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-01_overview', '/docs/about/01_overview', {})
        },
        {
            path: '/docs/about/02_introduction',
            meta: {
                h1: 'Coherence CLI Introduction',
                title: 'Coherence CLI Introduction',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-02_introduction', '/docs/about/02_introduction', {})
        },
        {
            path: '/docs/about/03_quickstart',
            meta: {
                h1: 'Quick Start',
                title: 'Quick Start',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-about-03_quickstart', '/docs/about/03_quickstart', {})
        },
        {
            path: '/docs/installation/01_installation',
            meta: {
                h1: 'Coherence CLI Installation',
                title: 'Coherence CLI Installation',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-installation-01_installation', '/docs/installation/01_installation', {})
        },
        {
            path: '/docs/config/01_overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-01_overview', '/docs/config/01_overview', {})
        },
        {
            path: '/docs/config/05_global_flags',
            meta: {
                h1: 'Global Flags',
                title: 'Global Flags',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-05_global_flags', '/docs/config/05_global_flags', {})
        },
        {
            path: '/docs/config/06_bytes_display_format',
            meta: {
                h1: 'Bytes Display Format',
                title: 'Bytes Display Format',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-06_bytes_display_format', '/docs/config/06_bytes_display_format', {})
        },
        {
            path: '/docs/config/07_command_completion',
            meta: {
                h1: 'Command Completion',
                title: 'Command Completion',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-07_command_completion', '/docs/config/07_command_completion', {})
        },
        {
            path: '/docs/config/09_get_config',
            meta: {
                h1: 'Get Config',
                title: 'Get Config',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-09_get_config', '/docs/config/09_get_config', {})
        },
        {
            path: '/docs/config/10_changing_config_locations',
            meta: {
                h1: 'Changing Config Locations',
                title: 'Changing Config Locations',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-config-10_changing_config_locations', '/docs/config/10_changing_config_locations', {})
        },
        {
            path: '/docs/reference/01_overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-01_overview', '/docs/reference/01_overview', {})
        },
        {
            path: '/docs/reference/05_clusters',
            meta: {
                h1: 'Clusters',
                title: 'Clusters',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-05_clusters', '/docs/reference/05_clusters', {})
        },
        {
            path: '/docs/reference/10_contexts',
            meta: {
                h1: 'Contexts',
                title: 'Contexts',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-10_contexts', '/docs/reference/10_contexts', {})
        },
        {
            path: '/docs/reference/15_members',
            meta: {
                h1: 'Members',
                title: 'Members',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-15_members', '/docs/reference/15_members', {})
        },
        {
            path: '/docs/reference/17_machines',
            meta: {
                h1: 'Machines',
                title: 'Machines',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-17_machines', '/docs/reference/17_machines', {})
        },
        {
            path: '/docs/reference/20_services',
            meta: {
                h1: 'Services',
                title: 'Services',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-20_services', '/docs/reference/20_services', {})
        },
        {
            path: '/docs/reference/25_caches',
            meta: {
                h1: 'Caches',
                title: 'Caches',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-25_caches', '/docs/reference/25_caches', {})
        },
        {
            path: '/docs/reference/30_topics',
            meta: {
                h1: 'Topics',
                title: 'Topics',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-30_topics', '/docs/reference/30_topics', {})
        },
        {
            path: '/docs/reference/40_persistence',
            meta: {
                h1: 'Persistence',
                title: 'Persistence',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-40_persistence', '/docs/reference/40_persistence', {})
        },
        {
            path: '/docs/reference/42_federation',
            meta: {
                h1: 'Federation',
                title: 'Federation',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-42_federation', '/docs/reference/42_federation', {})
        },
        {
            path: '/docs/reference/45_nslookup',
            meta: {
                h1: 'NS Lookup',
                title: 'NS Lookup',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-45_nslookup', '/docs/reference/45_nslookup', {})
        },
        {
            path: '/docs/reference/50_proxies',
            meta: {
                h1: 'Proxy Servers',
                title: 'Proxy Servers',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-50_proxies', '/docs/reference/50_proxies', {})
        },
        {
            path: '/docs/reference/55_http_servers',
            meta: {
                h1: 'Http Servers',
                title: 'Http Servers',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-55_http_servers', '/docs/reference/55_http_servers', {})
        },
        {
            path: '/docs/reference/56_elastic_data',
            meta: {
                h1: 'Elastic Data',
                title: 'Elastic Data',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-56_elastic_data', '/docs/reference/56_elastic_data', {})
        },
        {
            path: '/docs/reference/58_http_sessions',
            meta: {
                h1: 'Http Sessions',
                title: 'Http Sessions',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-58_http_sessions', '/docs/reference/58_http_sessions', {})
        },
        {
            path: '/docs/reference/60_executors',
            meta: {
                h1: 'Executors',
                title: 'Executors',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-60_executors', '/docs/reference/60_executors', {})
        },
        {
            path: '/docs/reference/66_reporters',
            meta: {
                h1: 'Reporters',
                title: 'Reporters',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-66_reporters', '/docs/reference/66_reporters', {})
        },
        {
            path: '/docs/reference/85_diagnostics',
            meta: {
                h1: 'Diagnostics',
                title: 'Diagnostics',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-85_diagnostics', '/docs/reference/85_diagnostics', {})
        },
        {
            path: '/docs/reference/90_health',
            meta: {
                h1: 'Health',
                title: 'Health',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-90_health', '/docs/reference/90_health', {})
        },
        {
            path: '/docs/reference/92_reset',
            meta: {
                h1: 'Resetting Statistics',
                title: 'Resetting Statistics',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-92_reset', '/docs/reference/92_reset', {})
        },
        {
            path: '/docs/reference/95_misc',
            meta: {
                h1: 'Miscellaneous',
                title: 'Miscellaneous',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-95_misc', '/docs/reference/95_misc', {})
        },
        {
            path: '/docs/reference/98_create_clusters',
            meta: {
                h1: 'Creating Clusters',
                title: 'Creating Clusters',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-reference-98_create_clusters', '/docs/reference/98_create_clusters', {})
        },
        {
            path: '/docs/security/01_overview',
            meta: {
                h1: 'Securing CLI Access',
                title: 'Securing CLI Access',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-security-01_overview', '/docs/security/01_overview', {})
        },
        {
            path: '/docs/troubleshooting/01_trouble-shooting',
            meta: {
                h1: 'Troubleshooting Guide',
                title: 'Troubleshooting Guide',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-troubleshooting-01_trouble-shooting', '/docs/troubleshooting/01_trouble-shooting', {})
        },
        {
            path: '/docs/examples/01_overview',
            meta: {
                h1: 'Overview',
                title: 'Overview',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-01_overview', '/docs/examples/01_overview', {})
        },
        {
            path: '/docs/examples/05_rolling_restarts',
            meta: {
                h1: 'Rolling Restarts',
                title: 'Rolling Restarts',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-05_rolling_restarts', '/docs/examples/05_rolling_restarts', {})
        },
        {
            path: '/docs/examples/10_jsonpath',
            meta: {
                h1: 'Using JSONPath',
                title: 'Using JSONPath',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-10_jsonpath', '/docs/examples/10_jsonpath', {})
        },
        {
            path: '/docs/examples/15_set_cache_attrs',
            meta: {
                h1: 'Setting Cache Attributes',
                title: 'Setting Cache Attributes',
                h1Prefix: null,
                description: null,
                keywords: null,
                customLayout: null,
                hasNav: true
            },
            component: loadPage('docs-examples-15_set_cache_attrs', '/docs/examples/15_set_cache_attrs', {})
        },
        {
            path: '/', redirect: '/docs/about/01_overview'
        },
        {
            path: '*', redirect: '/'
        }
    ];
}

function createNav(){
    return [
        { header: 'Core documentation' },
        {
            title: 'About',
            action: 'assistant',
            group: '/about',
            items: [
                { href: '/docs/about/01_overview', title: 'Overview' },
                { href: '/docs/about/02_introduction', title: 'Coherence CLI Introduction' },
                { href: '/docs/about/03_quickstart', title: 'Quick Start' }
            ]
        },
        {
            title: 'Installation',
            action: 'fa-save',
            group: '/installation',
            items: [
                { href: '/docs/installation/01_installation', title: 'Coherence CLI Installation' }
            ]
        },
        {
            title: 'Configuration',
            action: 'fa-cogs',
            group: '/config',
            items: [
                { href: '/docs/config/01_overview', title: 'Overview' },
                { href: '/docs/config/05_global_flags', title: 'Global Flags' },
                { href: '/docs/config/06_bytes_display_format', title: 'Bytes Display Format' },
                { href: '/docs/config/07_command_completion', title: 'Command Completion' },
                { href: '/docs/config/09_get_config', title: 'Get Config' },
                { href: '/docs/config/10_changing_config_locations', title: 'Changing Config Locations' }
            ]
        },
        {
            title: 'Command Reference',
            action: 'find_in_page',
            group: '/reference',
            items: [
                { href: '/docs/reference/01_overview', title: 'Overview' },
                { href: '/docs/reference/05_clusters', title: 'Clusters' },
                { href: '/docs/reference/10_contexts', title: 'Contexts' },
                { href: '/docs/reference/15_members', title: 'Members' },
                { href: '/docs/reference/17_machines', title: 'Machines' },
                { href: '/docs/reference/20_services', title: 'Services' },
                { href: '/docs/reference/25_caches', title: 'Caches' },
                { href: '/docs/reference/30_topics', title: 'Topics' },
                { href: '/docs/reference/40_persistence', title: 'Persistence' },
                { href: '/docs/reference/42_federation', title: 'Federation' },
                { href: '/docs/reference/45_nslookup', title: 'NS Lookup' },
                { href: '/docs/reference/50_proxies', title: 'Proxy Servers' },
                { href: '/docs/reference/55_http_servers', title: 'Http Servers' },
                { href: '/docs/reference/56_elastic_data', title: 'Elastic Data' },
                { href: '/docs/reference/58_http_sessions', title: 'Http Sessions' },
                { href: '/docs/reference/60_executors', title: 'Executors' },
                { href: '/docs/reference/66_reporters', title: 'Reporters' },
                { href: '/docs/reference/85_diagnostics', title: 'Diagnostics' },
                { href: '/docs/reference/90_health', title: 'Health' },
                { href: '/docs/reference/92_reset', title: 'Resetting Statistics' },
                { href: '/docs/reference/95_misc', title: 'Miscellaneous' },
                { href: '/docs/reference/98_create_clusters', title: 'Creating Clusters' }
            ]
        },
        {
            title: 'Security',
            action: 'lock',
            group: '/security',
            items: [
                { href: '/docs/security/01_overview', title: 'Securing CLI Access' }
            ]
        },
        {
            title: 'Troubleshooting',
            action: 'fa-question-circle',
            group: '/troubleshooting',
            items: [
                { href: '/docs/troubleshooting/01_trouble-shooting', title: 'Troubleshooting Guide' }
            ]
        },
        {
            title: 'Examples',
            action: 'explore',
            group: '/examples',
            items: [
                { href: '/docs/examples/01_overview', title: 'Overview' },
                { href: '/docs/examples/05_rolling_restarts', title: 'Rolling Restarts' },
                { href: '/docs/examples/10_jsonpath', title: 'Using JSONPath' },
                { href: '/docs/examples/15_set_cache_attrs', title: 'Setting Cache Attributes' }
            ]
        },
        { divider: true },
        { header: 'Additional resources' },
        {
            title: 'Slack',
            action: 'fa-slack',
            href: 'https://join.slack.com/t/oraclecoherence/shared_invite/enQtNzcxNTQwMTAzNjE4LTJkZWI5ZDkzNGEzOTllZDgwZDU3NGM2YjY5YWYwMzM3ODdkNTU2NmNmNDFhOWIxMDZlNjg2MzE3NmMxZWMxMWE',
            target: '_blank'
        },
        {
            title: 'Coherence Community',
            action: 'people',
            href: 'https://coherence.community',
            target: '_blank'
        },
        {
            title: 'GitHub',
            action: 'fa-github-square',
            href: 'https://github.com/oracle/coherence-cli',
            target: '_blank'
        }
    ];
}