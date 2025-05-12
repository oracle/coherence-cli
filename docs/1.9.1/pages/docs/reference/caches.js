<doc-view>

<h2 id="_caches">Caches</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage cluster caches.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-caches" @click.native="this.scrollFix('#get-caches')"><code>cohctl get caches</code></router-link> - displays the caches for a cluster</p>

</li>
<li>
<p><router-link to="#describe-cache" @click.native="this.scrollFix('#describe-cache')"><code>cohctl describe cache</code></router-link> - shows information related to a specific cache and service</p>

</li>
<li>
<p><router-link to="#get-cache-stores" @click.native="this.scrollFix('#get-cache-stores')"><code>cohctl get cache-stores</code></router-link> - displays cache stores for a cache and service</p>

</li>
<li>
<p><router-link to="#get-cache-storage" @click.native="this.scrollFix('#get-cache-storage')"><code>cohctl get cache-storage</code></router-link> - displays storage information for a cache and service</p>

</li>
<li>
<p><router-link to="#get-cache-access" @click.native="this.scrollFix('#get-cache-access')"><code>cohctl get cache-access</code></router-link> - displays access information for a cache and service</p>

</li>
<li>
<p><router-link to="#get-cache-indexes" @click.native="this.scrollFix('#get-cache-indexes')"><code>cohctl get cache-indexes</code></router-link> - displays cache index information for a cache and service</p>

</li>
<li>
<p><router-link to="#get-cache-partitions" @click.native="this.scrollFix('#get-cache-partitions')"><code>cohctl get cache-partitions</code></router-link> - displays partition information for a cache and service</p>

</li>
<li>
<p><router-link to="#set-cache" @click.native="this.scrollFix('#set-cache')"><code>cohctl set cache</code></router-link> - sets an attribute for a cache across one or more members</p>

</li>
<li>
<p><router-link to="#truncate-cache" @click.native="this.scrollFix('#truncate-cache')"><code>cohctl truncate cache</code></router-link> - truncates a caches contents, not generating any cache events</p>

</li>
<li>
<p><router-link to="#clear-cache" @click.native="this.scrollFix('#clear-cache')"><code>cohctl clear cache</code></router-link> - clears a caches contents</p>

</li>
</ul>


<h4 id="get-caches">Get Caches</h4>
<div class="section">
<p>The 'get caches' command displays caches for a cluster. If no service
name is specified then all services are queried. You can specify '-o wide' to
display addition information. Use '-I' to ignore internal caches such as those
used by Federation.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get caches [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for caches
  -I, --ignore-special   ignore system caches or caches with $
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all caches and display cache size in MB using <code>-m</code> option.</p>

<markup
lang="bash"

>cohctl get caches -c local -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>Total Caches: 2, Total primary storage: 36 MB

SERVICE           CACHE        COUNT   SIZE  AVG SIZE     PUTS    GETS  REMOVES    HITS  MISSES  HIT PROB
PartitionedCache  customers  100,000  25 MB       262  200,000       0        0       0       0     0.00%
PartitionedCache  orders      10,000  11 MB     1,160   20,000  20,000        0  20,000       0   100.00%</markup>

<div class="admonition note">
<p class="admonition-inline">If you do not use the <code>-o wide</code> option, then you will only see service, cache name, count and size.</p>
</div>

<p>Display all caches for a particular service.</p>

<markup
lang="bash"

>cohctl get caches -c local -s PartitionedCache -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>Total Caches: 2, Total primary storage: 36 MB

SERVICE           CACHE        COUNT   SIZE  AVG SIZE     PUTS    GETS  REMOVES    HITS  MISSES  HIT PROB
PartitionedCache  customers  100,000  25 MB       262  200,000       0        0       0       0     0.00%
PartitionedCache  orders      10,000  11 MB     1,160   20,000  20,000        0  20,000       0   100.00%</markup>

</div>


<h4 id="describe-cache">Describe Cache</h4>
<div class="section">
<p>The 'describe cache' command displays information related to a specific cache. This
includes cache size, access, storage and index information across all nodes.
You can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe cache cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe cache orders -c local -s PartitionedCache -m</markup>

<p>Output:</p>

<markup
lang="bash"

>CACHE DETAILS
-------------
Service         :  PartitionedCache
Name            :  orders
Type            :  Cache
Description     :  Implementation: com.tangosol.net.cache.LocalCache
Cache Store Type:  NONE

CACHE SIZE AND ACCESS DETAILS
-----------------------------
NODE ID  TIER   COUNT  SIZE    PUTS    GETS  REMOVES
      1  back   5,022  5 MB   5,022  10,044        0
      2  back   4,978  5 MB   4,978   9,956        0
      3  front      0  0 MB  10,000       0        0

CACHE STORAGE DETAILS
---------------------
NODE ID  TIER   LOCKS GRANTED  LOCKS PENDING  LISTENERS  MAX QUERY MS  MAX QUERY DESC
      1  back               0              0          0             0
      2  back               0              0          0             0
      3  front              0              0          0             0

INDEX DETAILS
-------------
Total Indexing Bytes:  2,720,664
Total Indexing:        2 MB
Total Indexing Millis: 263

Node:1:   SimpleMapIndex: Extractor=.toString(), Ordered=true, Footprint=784KB, Content=5022
          SimpleMapIndex: Extractor=.hashCode(), Ordered=true, Footprint=549KB, Content=5022
Node:2:   SimpleMapIndex: Extractor=.toString(), Ordered=true, Footprint=777KB, Content=4978
          SimpleMapIndex: Extractor=.hashCode(), Ordered=true, Footprint=544KB, Content=4978

CACHE STORE DETAILS
-------------------
Total Queue Size:     6,708,931
Total Store Failures: 0
Total Store Failures: 0

NODE ID  QUEUE SIZE  WRITES  AVG BATCH  AVG WRITE  TOTAL WRITE  FAILURES  READS  AVG READ  TOTAL READ
      1   2,261,151   8,042        127      266ms      35m 43s         0      0       0ms        0.0s
      2   2,222,822   7,966        127      266ms      35m 21s         0      0       0ms        0.0s
      3   2,224,958   7,937        127      266ms      35m 18s         0      0       0ms        0.0s</markup>

<div class="admonition note">
<p class="admonition-inline">You may omit the service name option if the cache name is unique.</p>
</div>

<div class="admonition note">
<p class="admonition-inline">You can also use the <code>-o wide</code> option to display more detailed information.</p>
</div>

<div class="admonition note">
<p class="admonition-inline">The default memory display format is bytes but can be changed by using <code>-k</code>, <code>-m</code> or <code>-g</code>.</p>
</div>

<markup
lang="bash"

>cohctl describe cache test -c local -s PartitionedCache -o wide -m</markup>

<p>Output:</p>

<markup
lang="bash"

>CACHE DETAILS
-------------
Service         :  PartitionedCache
Name            :  orders
Type            :  Cache
Description     :  Implementation: com.tangosol.net.cache.LocalCache
Cache Store Type:  NONE

CACHE SIZE AND ACCESS DETAILS
-----------------------------
NODE ID  TIER   COUNT  SIZE  TOTAL PUTS  TOTAL GETS  TOTAL REMOVES    HITS  MISSES  HIT PROB  STORE READS  WRITES  FAILURES
      1  back   5,022  5 MB       5,022      10,044              0  10,044       0   100.00%           -1      -1        -1
      2  back   4,978  5 MB       4,978       9,956              0   9,956       0   100.00%           -1      -1        -1
      3  front      0  0 MB      10,000           0              0       0       0     0.00%           -1      -1        -1

CACHE STORAGE DETAILS
---------------------
NODE ID  TIER   LOCKS GRANTED  LOCKS PENDING  LISTENERS  MAX QUERY MS  MAX QUERY DESC  NO OPT AVG  OPT AVG  INDEX SIZE  INDEXING MILLIS
      1  back               0              0          0             0                      0.0000   0.0000        1 MB              118
      2  back               0              0          0             0                      0.0000   0.0000        1 MB              145
      3  front              0              0          0             0                      0.0000   0.0000        0 MB                0

INDEX DETAILS
-------------
Total Indexing Bytes:  2,720,664
Total Indexing:        2 MB
Total Indexing Millis: 263

Node:1:   SimpleMapIndex: Extractor=.toString(), Ordered=true, Footprint=784KB, Content=5022
          SimpleMapIndex: Extractor=.hashCode(), Ordered=true, Footprint=549KB, Content=5022
Node:2:   SimpleMapIndex: Extractor=.toString(), Ordered=true, Footprint=777KB, Content=4978
          SimpleMapIndex: Extractor=.hashCode(), Ordered=true, Footprint=544KB, Content=4978

CACHE STORE DETAILS
-------------------
Total Queue Size:     6,708,931
Total Store Failures: 0
Total Store Failures: 0

NODE ID  QUEUE SIZE  WRITES  AVG BATCH  AVG WRITE  FAILURES  READS  AVG READ
      1   2,261,151   8,042        127      266ms         0      0       0ms
      2   2,222,822   7,966        127      266ms         0      0       0ms
      3   2,224,958   7,937        127      266ms         0      0       0ms</markup>

</div>


<h4 id="get-cache-stores">Get Cache Stores</h4>
<div class="section">
<p>The 'get cache-stores' command displays cache store information related to a specific cache.
You can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get cache-stores cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-stores
  -s, --service string   Service name</pre>
</div>

<markup
lang="bash"

>cohctl get cache-stores -c local test -s DistributedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>Service/Cache:        DistributedCache/test
Cache Store Type:     WRITE-BEHIND
Total Queue Size:     6,708,931
Total Store Failures: 0
Total Store Failures: 0

NODE ID  QUEUE SIZE  WRITES  AVG BATCH  AVG WRITE  FAILURES  READS  AVG READ
      1   2,261,151   8,042        127      266ms         0      0       0ms
      2   2,222,822   7,966        127      266ms         0      0       0ms
      3   2,224,958   7,937        127      266ms         0      0       0ms</markup>

<p>You may omit the service name if the cache name is unique.</p>

<div class="admonition note">
<p class="admonition-inline">If you do not use the <code>-o wide</code> option, then you will only see service, cache name, count and size.</p>
</div>

</div>


<h4 id="get-cache-storage">Get Cache Storage</h4>
<div class="section">
<p>The 'get cache-storage' command displays cache storage information related to a specific cache.
You can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get cache-storage cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-storage
  -q, --query            show query details in full
  -s, --service string   Service name</pre>
</div>

<markup
lang="bash"

>cohctl get cache-storage -c local test</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache: test

NODE ID  TIER  LOCKS GRANTED  LOCKS PENDING  KEY LISTENERS  FILTER LISTENERS  MAX QUERY MS  MAX QUERY DESC
      1  back              0              0              0                 0             0
      2  back              0              0              0                 0             0
      3  back              0              0              0                 0             0</markup>

</div>


<h4 id="get-cache-access">Get Cache Access</h4>
<div class="section">
<p>The 'get cache-access' command displays cache access information related to a specific cache.
You can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get cache-access cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-access
  -s, --service string   Service name</pre>
</div>

<markup
lang="bash"

>cohctl get cache-access -c local test</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache: test

NODE ID  TIER  COUNT  SIZE  PUTS  GETS  REMOVES  CLEARS  EVICTIONS
      1  back      0  0 MB     0     0        0       2          0
      2  back      0  0 MB     2     0        2       2          0
      3  back      0  0 MB     0     0        0       2          0</markup>

</div>


<h4 id="get-cache-indexes">Get Cache Indexes</h4>
<div class="section">
<p>The 'get cache-indexes' command displays cache index information related to a specific cache.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get cache-indexes cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-indexes
  -s, --service string   Service name</pre>
</div>

<markup
lang="bash"

>cohctl get cache-indexes -c local test</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache: test

Total Indexing Bytes:  49,756
Total Indexing:        0 MB
Total Indexing Millis: 1,942

Node:2:   PartitionedIndex: Extractor=.toString()(), Ordered=true, Footprint=16.2KB
Node:1:   PartitionedIndex: Extractor=.toString()(), Ordered=true, Footprint=16.2KB
Node:3:   PartitionedIndex: Extractor=.toString()(), Ordered=true, Footprint=16.0KB</markup>

<div class="admonition note">
<p class="admonition-inline">Depending upon the Coherence version, you may not see all the above information.</p>
</div>

</div>


<h4 id="get-cache-partitions">Get Cache Partitions</h4>
<div class="section">
<p>The 'get cache-partitions' command displays cache partition information related to a specific cache.
This information includes: partition id, count of entries and size of entries (binary key + value lengths).</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get cache-partitions cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache-partitions
  -s, --service string   Service name
  -S, --summary          show a cache summary</pre>
</div>

<p><strong>Examples</strong></p>

<p>Get a summary of partition information for a cache.</p>

<markup
lang="bash"

>cohctl get cache-partitions -c local test -S</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache:       test
Partitions:  257
Total Count: 200,000
Total Size:  116 MB</markup>

<p>Get all partition information for a cache.</p>

<markup
lang="bash"

>cohctl get cache-partitions -c local test</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache:       test
Partitions:  257
Total Count: 200,000
Total Size:  116 MB

PARTITION  COUNT  SIZE
        0    789  0 MB
        1    790  0 MB
        2    747  0 MB
        3    784  0 MB
        4    730  0 MB
...
      253    782  0 MB
      254    752  0 MB
      255    773  0 MB
      256    785  0 MB</markup>

<p>Get partition information and sort by size of cache in partition descending. (Using KB format)</p>

<markup
lang="bash"

>cohctl get cache-partitions test -c local -k --sort-size | head -20</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache:       test
Partitions:  257
Total Count: 200,000
Total Size:  119,706 KB

PARTITION  COUNT    SIZE
      159    852  509 KB
      229    841  503 KB
      140    836  500 KB
        5    833  498 KB
       25    828  495 KB
      203    826  494 KB
      145    821  491 KB
      179    821  491 KB
      235    819  490 KB
      199    819  490 KB
      193    818  489 KB
      188    817  488 KB</markup>

<p>Get partition information and sort by count of entries in partition descending. (Using KB format)</p>

<markup
lang="bash"

>cohctl get cache-partitions test -c local -k --sort-count | head -20</markup>

<p>Output:</p>

<markup
lang="bash"

>Cache:       test
Partitions:  257
Total Count: 200,000
Total Size:  119,706 KB

PARTITION  COUNT    SIZE
      159    852  509 KB
      229    841  503 KB
      140    836  500 KB
        5    833  498 KB
       25    828  495 KB
      203    826  494 KB
      179    821  491 KB
      145    821  491 KB
      199    819  490 KB
      235    819  490 KB
      193    818  489 KB
      188    817  488 KB</markup>

</div>


<h4 id="set-cache">Set Cache</h4>
<div class="section">
<p>The 'set cache' command sets an attribute for a cache across one or member nodes.
The following attribute names are allowed: expiryDelay, highUnits, lowUnits,
batchFactor, refreshFactor or requeueThreshold.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set cache cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -a, --attribute string   attribute name to set
  -h, --help               help for cache
  -n, --node string        comma separated node ids to target (default "all")
  -s, --service string     Service name
  -t, --tier string        tier to apply to, back or front (default "back")
  -v, --value string       attribute value to set
  -y, --yes                automatically confirm the operation</pre>
</div>

<p>See the <a target="_blank" href="https://docs.oracle.com/en/middleware/fusion-middleware/coherence/14.1.2/manage/oracle-coherence-mbeans-reference.html">Cache MBean Reference</a>
for more information on the above attributes.</p>

<p><strong>Examples</strong></p>

<p>Set the expiry delay for all nodes to 30 seconds. (we are assuming we have 3 storage-enabled nodes)</p>

<markup
lang="bash"

>cohctl set cache test -a expiryDelay -v 30 -s PartitionedCache -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Selected service/cache: PartitionedCache/test
Are you sure you want to set the value of attribute expiryDelay to 30 in tier back for all 3 nodes? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">See <router-link to="/docs/examples/set_cache_attrs">here</router-link> for a more detailed example of this command.</p>
</div>

</div>


<h4 id="truncate-cache">Truncate Cache</h4>
<div class="section">
<p>The 'truncate cache' command issues a truncate against a specific cache. The truncate cache will not generate cache events.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl truncate cache cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>cohctl truncate cache test -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to truncate cache test in service PartitionedCache? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">You may omit the service name option if the cache name is unique.</p>
</div>

</div>


<h4 id="clear-cache">Clear Cache</h4>
<div class="section">
<p>The 'clear cache' command issues a clear against a specific cache.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl clear cache cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for cache
  -s, --service string   Service name
  -y, --yes              automatically confirm the operation</pre>
</div>

<markup
lang="bash"

>cohctl clear cache test -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Are you sure you want to clear cache test in service PartitionedCache? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">You may omit the service name option if the cache name is unique.</p>
</div>

</div>

</div>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/services">Services</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/topics">Topics</router-link></p>

</li>
</ul>

</div>

</div>

</doc-view>
