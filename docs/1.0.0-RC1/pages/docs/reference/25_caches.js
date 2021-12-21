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
<p><router-link to="#set-cache" @click.native="this.scrollFix('#set-cache')"><code>cohctl set cache</code></router-link> - sets an attribute for a cache across one or more members</p>

</li>
</ul>

<h4 id="get-caches">Get Caches</h4>
<div class="section">
<p>The 'get caches' command displays caches for a cluster. If
no service name is specified then all services are queried. You
can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get caches [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for caches
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all caches.</p>

<markup
lang="bash"

>$ cohctl get caches -c local
Total Caches: 3, Total primary storage: 16MB

SERVICE            CACHE     CACHE SIZE       BYTES    MB  AVG SIZE  TOTAL PUTS  TOTAL GETS  TOTAL REMOVES  TOTAL HITS  TOTAL MISSES  HIT PROB
PartitionedCache   test1        100,000  16,800,000  16MB       168     200,000           0              0           0             0     0.00%
PartitionedCache   test2            123     142,680   0MB     1,160         246           0              0           0             0     0.00%
PartitionedCache2  test-123           0           0   0MB         0           0           0              0           0             0     0.00%</markup>

<p>Display all caches for a particular service.</p>

<markup
lang="bash"

>$ cohctl get caches -c local -s PartitionedCache2

Total Caches: 1, Total primary storage: 0MB

SERVICE            CACHE     CACHE SIZE  BYTES   MB  AVG SIZE  TOTAL PUTS  TOTAL GETS  TOTAL REMOVES  TOTAL HITS  TOTAL MISSES  HIT PROB
PartitionedCache2  test-123           0      0  0MB         0           0           0              0           0             0     0.00%</markup>

</div>

<h4 id="describe-cache">Describe Cache</h4>
<div class="section">
<p>The 'describe cache' command displays information related to a specific cache. This
includes cache size, access, storage and index information across all nodes. You
can specify '-o wide' to display addition information.</p>

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

>$ cohctl describe cache test -c local -s PartitionedCache</markup>

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
  -y, --yes                Automatically confirm the operation</pre>
</div>

<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.0/manage/oracle-coherence-mbeans-reference.html">Cache MBean Reference</a>
for more information on the above attributes.</p>

<p><strong>Examples</strong></p>

<p>Set the expiry delay for all nodes to 30 seconds. (we are assuming we have 3 storage-enabled nodes)</p>

<markup
lang="bash"

>$ cohctl set cache test -a expiryDelay -v 30 -s PartitionedCache
Using cluster connection 'local' from current context.

Selected service/cache: PartitionedCache/test
Are you sure you want to set the value of attribute expiryDelay to 30 in tier back for all 3 nodes? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">See <router-link to="/docs/examples/15_set_cache_attrs">here</router-link> for a more detailed example of this command.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/30_topics">Topics</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
