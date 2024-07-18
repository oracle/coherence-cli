<doc-view>

<h2 id="_monitor_clusters">Monitor Clusters</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>Experimental command to monitor clusters with text UI to show multiple panels.</p>

<div class="admonition note">
<p class="admonition-inline">The <code>monitor cluster</code> command is currently experimental only and may be changed or removed in the future</p>
</div>
<ul class="ulist">
<li>
<p><router-link to="#monitor-cluster" @click.native="this.scrollFix('#monitor-cluster')"><code>cohctl monitor cluster</code></router-link> - monitors the cluster using text based UI</p>

</li>
<li>
<p><router-link to="#monitor-cluster-panels" @click.native="this.scrollFix('#monitor-cluster-panels')"><code>cohctl monitor cluster --show-panels</code></router-link> - shows all available panels</p>

</li>
</ul>

<h4 id="monitor-cluster">Monitor Cluster</h4>
<div class="section">
<p>The 'monitor cluster' command displays a text based UI to monitor the overall cluster.
You can specify a layout to show by providing a value for '-l'. Panels can be specified using 'panel1:panel1,panel3'.
Specifying a ':' is the line separator and ',' means panels on the same line. If you don&#8217;t specify one the 'default' layout is used.
There are a number of layouts available: 'default-service', 'default-cache', 'default-topic' and 'default-subscriber' which
require you to specify cache, service, topic or subscriber.
Use --show-panels to show all available panels.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl monitor cluster connection-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -C, --cache-name string   cache name
  -h, --help                help for cluster
  -I, --ignore-errors       ignore errors after initial refresh
  -l, --layout string       layout to use (default "default")
  -S, --service string      Service name
      --show-panels         show all available panels
  -B, --subscriber-id int   subscriber
  -T, --topic-name string   topic name</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">You can also use <code>-o wide</code> to get wide output.</p>
</div>
<p><strong>Examples</strong></p>

<p><strong>Monitor a cluster using the default layout.</strong></p>

<markup
lang="bash"

>cohctl monitor cluster local</markup>

<p>Output:</p>

<markup
lang="bash"

> Coherence CLI: 2024-05-06 13:25:17 - Monitoring cluster local (22.06.8) ESC to quit (press key in [] to toggle expand, ? = help).
  (75.289463ms)
┌─Members [1]─(trimmed)──────────────────────────────────────────┐┌─Health Summary [2]────────────────────────────────────────────┐
│Total cluster members: 3                                        ││NAME                  SUB TYPE   MEMBERS  STARTED  LIVE  READY │
│Storage enabled count: 3                                        ││$SYS:Config           Service          3        3     3      3 │
│Departure count:       0                                        ││$SYS:HealthHttpProxy  Service          3        3     3      3 │
│                                                                ││$SYS:SystemProxy      Service          3        3     3      3 │
│Cluster Heap - Total: 768 MB Used: 221 MB Available: 547 MB (71.││Default               Coherence        3        3     3      3 │
│Storage Heap - Total: 768 MB Used: 221 MB Available: 547 MB (71.││ManagementHttpProxy   Service          1        1     1      1 │
│                                                                ││PartitionedCache      Service          3        3     3      3 │
│NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE            ││PartitionedTopic      Service          3        3     3      3 │
│      1  /127.0.0.1  50362    42980  storage-1  CoherenceServer ││Proxy                 Service          3        3     3      3 │
│      2  /127.0.0.1  50363    42981  storage-2  CoherenceServer ││                                                               │
└────────────────────────────────────────────────────────────────┘└───────────────────────────────────────────────────────────────┘
┌─Services [3]─(trimmed)─────────────────────────────────────────┐┌─Caches [4]────────────────────────────────────────────────────┐
│SERVICE NAME            TYPE              MEMBERS  STATUS HA  ST││                                                               │
│"$SYS:Config"           DistributedCache        3  NODE-SAFE    ││  No Content                                                   │
│"$SYS:HealthHttpProxy"  Proxy                   3  n/a          ││                                                               │
│"$SYS:SystemProxy"      Proxy                   3  n/a          ││                                                               │
│ManagementHttpProxy     Proxy                   1  n/a          ││                                                               │
│PartitionedCache        DistributedCache        3  NODE-SAFE    ││                                                               │
│PartitionedTopic        PagedTopic              3  NODE-SAFE    ││                                                               │
│Proxy                   Proxy                   3  n/a          ││                                                               │
└────────────────────────────────────────────────────────────────┘└───────────────────────────────────────────────────────────────┘
┌─Proxy Servers [5]──────────────────────────────────────────────┐┌─HTTP Servers [6]──────────────────────────────────────────────┐
│NODE ID  HOST IP              SERVICE NAME        CONNECTIONS  D││NODE ID  HOST IP        SERVICE NAME            SERVER TYPE    │
│1        0.0.0.0:50362.40119  "$SYS:SystemProxy"            0   ││1        0.0.0.0:50402  "$SYS:HealthHttpProxy"  com.tangosol.co│
│2        0.0.0.0:50363.49866  "$SYS:SystemProxy"            0   ││2        0.0.0.0:50401  "$SYS:HealthHttpProxy"  com.tangosol.co│
│3        0.0.0.0:50364.59927  "$SYS:SystemProxy"            0   ││3        0.0.0.0:50406  "$SYS:HealthHttpProxy"  com.tangosol.co│
│1        0.0.0.0:50362.34525  Proxy                         0   ││3        0.0.0.0:30000  ManagementHttpProxy     com.tangosol.co│
│2        0.0.0.0:50363.58603  Proxy                         0   ││                                                               │
│3        0.0.0.0:50364.55445  Proxy                         0   ││                                                               │
│                                                                ││                                                               │
└────────────────────────────────────────────────────────────────┘└───────────────────────────────────────────────────────────────┘
┌─Network Stats [7]───────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             PKT SENT  PKT REC  RESENT  EFFICIENCY  SEND Q  DATA SENT  DATA R│
│      1  /127.0.0.1  50362    42980  storage-1  CoherenceServer       259      314       2     100.00%       0       0 MB      0 │
│      2  /127.0.0.1  50363    42981  storage-2  CoherenceServer       141      108       1     100.00%       0       0 MB      0 │
│      3  /127.0.0.1  50364    42979  storage-0  CoherenceServer       149      113       0     100.00%       0       0 MB      0 │
│                                                                                                                                 │
│                                                                                                                                 │
│                                                                                                                                 │
│                                                                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘</markup>

<p>You can press <code>?</code> to display the help which is shown below:</p>

<markup
lang="bash"

> Monitor Cluster CLI Help

 - 'p' to toggle panel row padding
 - '+' to increase max height of all panels
 - '-' to decrease max height of all panels
 - '0' to reset max height of all panels
 - Key in [] to expand that panel
 - ESC / CTRL-C to exit monitoring

Press any key to exit help.</markup>

<div class="admonition note">
<p class="admonition-inline">If the title of a panel includes "trimmed" it means there are more rows to display.
You can press the key indicated in the <code>[]</code> to expand that panel.</p>
</div>
<p><strong>Monitor the cluster and specify the panels for services and caches on one line and then members on the next</strong></p>

<markup
lang="bash"

>cohctl monitor cluster local -l services,caches:members</markup>

<p>Output:</p>

<markup
lang="bash"

>Coherence CLI: 2024-05-06 13:26:47 - Monitoring cluster local (22.06.8) ESC to quit (press key in [] to toggle expand, ? = help).

┌─Services [1]─(trimmed)─────────────────────────────────────────┐┌─Caches [2]────────────────────────────────────────────────────┐
│SERVICE NAME            TYPE              MEMBERS  STATUS HA  ST││Total Caches: 3, Total primary storage: 4 MB                   │
│"$SYS:Config"           DistributedCache        3  NODE-SAFE    ││                                                               │
│"$SYS:HealthHttpProxy"  Proxy                   3  n/a          ││SERVICE           CACHE  COUNT  SIZE                           │
│"$SYS:SystemProxy"      Proxy                   3  n/a          ││PartitionedCache  test1    303  0 MB                           │
│ManagementHttpProxy     Proxy                   1  n/a          ││PartitionedCache  test2     30  0 MB                           │
│PartitionedCache        DistributedCache        4  NODE-SAFE    ││PartitionedCache  test3  4,004  4 MB                           │
│PartitionedTopic        PagedTopic              3  NODE-SAFE    ││                                                               │
│Proxy                   Proxy                   3  n/a          ││                                                               │
└────────────────────────────────────────────────────────────────┘└───────────────────────────────────────────────────────────────┘
┌─Members [3]─(trimmed)───────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│Total cluster members: 4                                                                                                         │
│Storage enabled count: 3                                                                                                         │
│Departure count:       0                                                                                                         │
│                                                                                                                                 │
│Cluster Heap - Total: 896 MB Used: 259 MB Available: 637 MB (71.1%)                                                              │
│Storage Heap - Total: 768 MB Used: 237 MB Available: 531 MB (69.1%)                                                              │
│                                                                                                                                 │
│NODE ID  ADDRESS     PORT   PROCESS  MEMBER                         ROLE              STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP   │
│      1  /127.0.0.1  50362    42980  storage-1                      CoherenceServer   true       256 MB      44 MB      212 MB   │
│      2  /127.0.0.1  50363    42981  storage-2                      CoherenceServer   true       256 MB      45 MB      211 MB   │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘----</markup>

<p><strong>Monitor the cluster and specify the <code>default-cache</code> layout and specify the cache <code>test1</code></strong></p>

<markup
lang="bash"

>cohctl monitor cluster local -l default-cache -C test1</markup>

<p>Output:</p>

<markup
lang="bash"

>Coherence CLI: 2024-05-06 11:13:59 - Monitoring cluster local (22.06.8) ESC to quit (press key in [] to toggle expand, ? = help).

┌─Caches [1]─────────────────────────────────────────────────────┐┌─Cache Indexes (PartitionedCache/test1) [2]────────────────────┐
│Total Caches: 3, Total primary storage: 4 MB                    ││Total Indexing Bytes:  0                                       │
│                                                                ││Total Indexing:        0 MB                                    │
│SERVICE           CACHE  COUNT  SIZE                            ││Total Indexing Millis: 0                                       │
│PartitionedCache  test1    303  0 MB                            │└───────────────────────────────────────────────────────────────┘
│PartitionedCache  test2     30  0 MB                            │
│PartitionedCache  test3  4,004  4 MB                            │
└────────────────────────────────────────────────────────────────┘
┌─Cache Access (PartitionedCache/test1) [3]───────────────────────────────────────────────────────────────────────────────────────┐
│NODE ID  TIER   COUNT  SIZE  PUTS   GETS  REMOVES  CLEARS  EVICTIONS                                                             │
│      1  back     102  0 MB   136  2,142        0       0          0                                                             │
│      2  back     103  0 MB   135  2,163        0       0          0                                                             │
│      3  back      98  0 MB   132  2,058        0       0          0                                                             │
│      4  front      0  0 MB   403      0        0       0          0                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
┌─Cache Storage (PartitionedCache/test1) [4]──────────────────────────────────────────────────────────────────────────────────────┐
│NODE ID  TIER   LOCKS GRANTED  LOCKS PENDING  KEY LISTENERS  FILTER LISTENERS  MAX QUERY MS  MAX QUERY DESC                      │
│      1  back               0              0              0                 0             0                                      │
│      2  back               0              0              0                 0             0                                      │
│      3  back               0              0              0                 0             0                                      │
│      4  front              0              0              0                 0             0                                      │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
┌─Cache Partitions (PartitionedCache/test1) [5]─(trimmed)─────────────────────────────────────────────────────────────────────────┐
│Partitions:       167                                                                                                            │
│Total Count:      303                                                                                                            │
│Total Size:       0 MB                                                                                                           │
│Max Entry Size:   1,160 (bytes)                                                                                                  │
│Owning Partition: 0                                                                                                              │
│                                                                                                                                 │
│PARTITION  OWNING MEMBER  COUNT  SIZE  MAX ENTRY SIZE                                                                            │
│        0              3      1  0 MB           1,160                                                                            │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘</markup>

<div class="admonition note">
<p class="admonition-inline">Any of the panels or layouts that specify <code>cache-<strong></code> or <code>service-</strong></code> must have the cache or service specified using
<code>-C</code> or <code>-S</code> respectively.</p>
</div>
</div>

<h4 id="monitor-cluster-panels">Monitor Cluster Show Panels</h4>
<div class="section">
<markup
lang="bash"

>cohctl monitor cluster --show-panels</markup>

<p>Output:</p>

<div class="listing">
<pre>Default panels
--------------
default-cache         : caches,cache-indexes:cache-access:cache-storage:cache-stores:cache-partitions
default-topic         : topics:topic-members:subscribers:subscriber-groups
default-subscriber    : topics:subscribers:subscriber-channels
default               : members,healthSummary:services,caches:proxies,http-servers:machines,network-stats
default-service       : services:service-members:service-distributions

Individual panels
-----------------
caches                : show caches
cache-access          : show cache access
cache-indexes         : show cache indexes
cache-storage         : show cache storage
cache-stores          : show cache stores
cache-partitions      : show cache partitions
departedMembers       : show departed members
elastic-data          : show elastic data
executors             : show Executors
healthSummary         : show health summary
federation-all        : show all federation details
federation-dest       : show federation destinations
federation-origins    : show federation origins
http-servers          : show HTTP servers
http-sessions         : show HTTP sessions
machines              : show machines
membersSummary        : show members summary
members               : show members
membersShort          : show members (short)
network-stats         : show network stats
persistence           : show persistence
proxies               : show proxy servers
proxy-connections     : show proxy connections
reporters             : show reporters
services              : show services
service-members       : show service members
service-distributions : show service distributions
service-storage       : show service storage
topic-members         : show topic members
subscribers           : show topic subscribers
subscriber-channels   : show topic subscriber channels
subscriber-groups     : show subscriber groups
topics                : show topics
view-caches           : show view caches</pre>
</div>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/05_clusters">Clusters</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
