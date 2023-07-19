<doc-view>

<h2 id="_resetting_statistics">Resetting Statistics</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>This section contains commands for resetting MBean statistics which can be
useful when you are running performance tests.</p>

<p>For most commands you can reset for all members or specify a comma-separated list of members
using the <code>-m</code> option.</p>

<p>See <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/oracle-coherence-mbeans-reference.html">MBean Reference</a>
for details on what MBeans support the <code>resetStatistics</code> operation.</p>

<div class="admonition note">
<p class="admonition-inline">Only the most recent Coherence versions support all the commands below and you be shown
a message if the operation is no supported for your specific Coherence version.</p>
</div>
<ul class="ulist">
<li>
<p><router-link to="#reset-cache-stats" @click.native="this.scrollFix('#reset-cache-stats')"><code>cohctl reset cache-stats</code></router-link> - resets cache statistics for all cache members or specific cache members</p>

</li>
<li>
<p><router-link to="#reset-executor-stats" @click.native="this.scrollFix('#reset-executor-stats')"><code>cohctl reset executor-stats</code></router-link> - resets statistics for an executor</p>

</li>
<li>
<p><router-link to="#reset-federation-stats" @click.native="this.scrollFix('#reset-federation-stats')"><code>cohctl reset federation-stats</code></router-link> - resets federation statistics for all federation or specific federation members</p>

</li>
<li>
<p><router-link to="#reset-flashjournal-stats" @click.native="this.scrollFix('#reset-flashjournal-stats')"><code>cohctl reset flashjournal-stats</code></router-link> - resets statistics for all flash journals</p>

</li>
<li>
<p><router-link to="#reset-ramjournal-stats" @click.native="this.scrollFix('#reset-ramjournal-stats')"><code>cohctl reset ramjournal-stats</code></router-link> - resets statistics for all ram journals</p>

</li>
<li>
<p><router-link to="#reset-ramjournal-stats" @click.native="this.scrollFix('#reset-ramjournal-stats')"><code>cohctl reset ramjournal-stats</code></router-link> - resets statistics for all ram journals</p>

</li>
<li>
<p><router-link to="#reset-member-stats" @click.native="this.scrollFix('#reset-member-stats')"><code>cohctl reset member-stats</code></router-link> - resets statistics for all or a specific member</p>

</li>
<li>
<p><router-link to="#reset-reporter-stats" @click.native="this.scrollFix('#reset-reporter-stats')"><code>cohctl reset reporter-stats</code></router-link> - resets reporter statistics for all or a specific reporter</p>

</li>
<li>
<p><router-link to="#reset-service-stats" @click.native="this.scrollFix('#reset-service-stats')"><code>cohctl reset service-stats</code></router-link> - resets services statistics for all service members or specific service members</p>

</li>
<li>
<p><router-link to="#reset-proxy-stats" @click.native="this.scrollFix('#reset-proxy-stats')"><code>cohctl reset proxy-stats</code></router-link> - resets proxy connection manager statistics for all proxy members or specific proxy members</p>

</li>
</ul>

<h4 id="reset-cache-stats">Reset Cache Statistics</h4>
<div class="section">
<p>The 'reset cache-stats' command resets cache statistics for all cache members or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset cache-stats cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-stats
  -n, --node string      comma separated node ids to target (default "all")
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Reset statistics for all cache members for cache <code>test</code>.</p>

<markup
lang="bash"

>$ cohctl get caches -c local

Total Caches: 1, Total primary storage: 30 MB

SERVICE           CACHE    COUNT   SIZE
PartitionedCache  test   123,000  30 MB

$ cohctl reset cache-stats test -s PartitionedCache
Using cluster connection 'local' from current context.

Are you sure you want to reset cache statistics for cache test, service PartitionedCache for all 3 nodes? (y/n) y
operation completed</markup>

<p>Reset statistics for cache members 1 and 2 for cache <code>test</code>.</p>

<markup
lang="bash"

>$ cohctl reset cache-stats test -s PartitionedCache -n 1,2 -c local

Are you sure you want to reset cache statistics for cache test, service PartitionedCache for 2 node(s)? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-executor-stats">Reset Executor Statistics</h4>
<div class="section">
<p>The 'reset executor-stats' command resets executor statistics for a specific executor.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset executor-stats executor-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for executor-stats
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get executors -c local

Total executors: 1
Running tasks:   0
Completed tasks: 0

NAME                                   MEMBER COUNT  IN PROGRESS  COMPLETED  REJECTED  DESCRIPTION
coherence-concurrent-default-executor             3            0          0         0  SingleThreaded(ThreadFactory=default)
coherence-cli$ (rc)$ cohctl reset executor-stats coherence-concurrent-default-executor
Using cluster connection 'local' from current context.

Are you sure you want to reset executor statistics for exeutor coherence-concurrent-default-executor? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-federation-stats">Reset Federation Statistics</h4>
<div class="section">
<p>The 'reset federation-stats' command resets federation statistics for all members or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset federation-stats service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help                 help for federation-stats
  -n, --node string          comma separated node ids to target (default "all")
  -p, --participant string   participant to apply to (default "all")
  -T, --type string          type to describe outgoing or incoming (default "outgoing")
  -y, --yes                  automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get federation all -c local

SERVICE           DESTINATION  MEMBERS  STATES  DATA SENT  MSG SENT  REC SENT  CURR AVG BWIDTH
FederatedService  cluster2           1  [IDLE]       3 MB    13,873    30,000          0.0Mbps

$ cohctl reset federation-stats FederatedService -p cluster2 -T outgoing -c local

Are you sure you want to reset federation statistics for service FederatedService, participant cluster2, type outgoing for all 1 nodes? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">The above federation command is only available in 14.1.1.2206.x and above.</p>
</div>
</div>

<h4 id="reset-ramjournal-stats">Reset RAM Journal Statistics</h4>
<div class="section">
<p>The 'reset ramjournal-stats' command resets ram journal statistics.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset ramjournal-stats [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for ramjournal-stats
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl reset ramjournal-stats -c local

Are you sure you want to reset ramjournal statistics for all 2 nodes? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-flashjournal-stats">Reset Flash Journal Statistics</h4>
<div class="section">
<p>The 'reset flashjournal-stats' command resets flash journal statistics.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset flashjournal-stats [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for flashjournal-stats
  -y, --yes    automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl reset flashjournal-stats -c local

Are you sure you want to reset flashjournal statistics for all 2 nodes? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-member-stats">Reset Member Statistics</h4>
<div class="section">
<p>The 'reset member-stats' command resets member statistics for all or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset member-stats [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for member-stats
  -n, --node string   comma separated node ids to target (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<p><strong>Examples</strong></p>

<p>Reset member statistics for member 1 and 3.</p>

<markup
lang="bash"

>$ cohctl get members -c local

Total cluster members: 3
Cluster Heap - Total: 384 MB Used: 127 MB Available: 257 MB (66.9%)
Storage Heap - Total: 384 MB Used: 127 MB Available: 257 MB (66.9%)

NODE ID  ADDRESS     PORT   PROCESS  MEMBER     ROLE             STORAGE  MAX HEAP  USED HEAP  AVAIL HEAP
      1  /127.0.0.1  58938    96295  storage-0  CoherenceServer  true       128 MB      65 MB       63 MB
      2  /127.0.0.1  58944    96303  storage-2  CoherenceServer  true       128 MB      41 MB       87 MB
      3  /127.0.0.1  58941    96296  storage-1  CoherenceServer  true       128 MB      21 MB      107 MB
$ cohctl reset member-stats -n 1,3 -c local

Are you sure you want to reset members statistics for 2 node(s)? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-reporter-stats">Reset Reporter Statistics</h4>
<div class="section">
<p>The 'reset reporter-stats' command resets reporter statistics for all or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset reporter-stats [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for reporter-stats
  -n, --node string   comma separated node ids to target (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>$ cohctl get reporters -c local

NODE ID  STATE    CONFIG FILE               OUTPUT PATH                       BATCH#  LAST REPORT  LAST RUN   AVG RUN  INTERVAL  AUTOSTART
      1  Stopped  reports/report-group.xml  /Users/user/Documents/Coheren...       0                    0ms  0.0000ms        60  false
      2  Stopped  reports/report-group.xml  /Users/user/Documents/Coheren...       0                    0ms  0.0000ms        60  false
      3  Stopped  reports/report-group.xml  /Users/user/Documents/Coheren...       0                    0ms  0.0000ms        60  false
$ cohctl reset reporter-stats -c local

Are you sure you want to reset reporters statistics for all 3 nodes? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-service-stats">Reset Service Statistics</h4>
<div class="section">
<p>The 'reset service-stats' command resets service statistics for all service or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset service-stats service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for service-stats
  -n, --node string   comma separated node ids to target (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>$ cohctl get services -c  local

SERVICE NAME            TYPE              MEMBERS  STATUS HA  STORAGE  PARTITIONS
Proxy                   Proxy                   3  n/a             -1          -1
PartitionedTopic        PagedTopic              3  NODE-SAFE        3         257
PartitionedCache        DistributedCache        3  NODE-SAFE        3         257
ManagementHttpProxy     Proxy                   1  n/a             -1          -1
"$SYS:HealthHttpProxy"  Proxy                   3  n/a             -1          -1
"$SYS:Config"           DistributedCache        3  NODE-SAFE        3         257
"$SYS:ConcurrentProxy"  Proxy                   3  n/a             -1          -1
"$SYS:Concurrent"       DistributedCache        3  NODE-SAFE        3         257

$ cohctl reset service-stats PartitionedCache -c local

Are you sure you want to reset service statistics for all 3 nodes? (y/n) y
operation completed</markup>

</div>

<h4 id="reset-proxy-stats">Reset Proxy Connection Manager Statistics</h4>
<div class="section">
<p>The 'reset proxy-stats' command resets connection manager statistics for all proxy services or a comma separated list of member IDs.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl reset proxy-stats service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help          help for proxy-stats
  -n, --node string   comma separated node ids to target (default "all")
  -y, --yes           automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>$ cohctl get proxies -c local

NODE ID  HOST IP              SERVICE NAME        CONNECTIONS  DATA SENT  DATA REC
1        0.0.0.0:64073.58994  "$SYS:SystemProxy"            0       0 MB      0 MB
1        0.0.0.0:64073.41509  Proxy                         0       0 MB      0 MB

$ cohctl reset proxy-stats Proxy -c local

Are you sure you want to reset connectionManager statics statistics for service Proxy for all 1 nodes? (y/n) y
operation completed</markup>

</div>
</div>
</div>
</doc-view>
