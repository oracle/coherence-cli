<doc-view>

<h2 id="_setting_cache_attributes">Setting Cache Attributes</h2>
<div class="section">
<p>This example shows you how to set various attributes for a cache at runtime.  Only the <strong>settable</strong> attributes
such as the following can be modified: expiryDelay, highUnits, lowUnits, batchFactor, refreshFactor and requeueThreshold.</p>

<p>See the <a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/oracle-coherence-mbeans-reference.html#GUID-A443DF50-F151-4E9B-AFC9-DFEDF4B149E7">Cache MBean Reference</a>
for more information on the above attributes.</p>

<p>In the example below, we will set the <code>expiryDelay</code> for a cache.</p>

<div class="admonition note">
<p class="admonition-inline">If you restart a cache node after setting an attribute, it will be reset to whatever the value was in
the cache configuration.</p>
</div>

<h3 id="_1_list_the_members_and_caches">1. List the members and caches</h3>
<div class="section">
<p>In this example we have a cluster of 3 storage-enabled members and a cache called <code>test</code>. Our context has been set to
<code>local</code> for our local cluster.</p>

<markup
lang="bash"

># Get the members
cohctl get members</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Total cluster members: 3
Cluster Heap - Total: 1.500GB, Used: 394MB, Available: 1.115GB (74.3%)

NODE ID  ADDRESS                      PORT  PROCESS  MEMBER  ROLE                  MAX HEAP  USED HEAP  AVAIL HEAP
      1  hostname-mac/192.168.1.124  60172    77425  n/a     Management               512MB       44MB       468MB
      2  hostname-mac/192.168.1.124  60178    77469  n/a     TangosolNetCoherence     512MB      190MB       322MB
      3  hostname-mac/192.168.1.124  60175    77447  n/a     TangosolNetCoherence     512MB      160MB       352MB</markup>

<p>Get the caches for the PartitionedCache service</p>

<markup
lang="bash"

>cohctl get caches -s PartitionedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Total Caches: 1, Total primary storage: 0MB

SERVICE           CACHE  CACHE SIZE  BYTES   MB
PartitionedCache  test            0      0  0MB</markup>

</div>

<h3 id="_2_use_jsonpath_to_display_the_current_expirydelay">2. Use JsonPath to display the current expiryDelay</h3>
<div class="section">
<p>Use the following to retrieve the expiry delay and nodes for the cache test.</p>

<markup
lang="bash"

>cohctl get caches -o jsonpath="$.items[?(@.name == 'test')]..['name','expiryDelay','nodeId']" |jq</markup>

<p>Output:</p>

<markup
lang="bash"

>[
  "test",
  [
    0
  ],
  [
    "1",
    "2",
    "3"
  ]
]</markup>

<div class="admonition note">
<p class="admonition-inline">You will see only 1 value of <code>0</code> for expiry delay because this query returns the distinct values.</p>
</div>
</div>

<h3 id="_3_set_the_expiry_delay_for_all_nodes_to_30_seconds">3. Set the expiry delay for all nodes to 30 seconds</h3>
<div class="section">
<p>The default tier is <code>back</code> and can be changed using the <code>-t</code> option to <code>front</code> if required.</p>

<markup
lang="bash"

>cohctl set cache test -a expiryDelay -v 30 -s PartitionedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Selected service/cache: PartitionedCache/test
Are you sure you want to set the value of attribute expiryDelay to 30 in tier back for all 3 nodes? (y/n) y
operation completed</markup>

<div class="admonition note">
<p class="admonition-inline">You will now see a value of <code>30</code> for all nodes.</p>
</div>
</div>

<h3 id="_4_re_query_the_expiry_delay">4. Re-query the expiry delay</h3>
<div class="section">
<markup
lang="bash"

>cohctl get caches -o jsonpath="$.items[?(@.name == 'test')]..['name','expiryDelay','nodeId']" |jq</markup>

<p>Output:</p>

<markup
lang="bash"

>[
  "test",
  [
    30
  ],
  [
    "1",
    "2",
    "3"
  ]
]</markup>

</div>

<h3 id="_5_set_the_expiry_delay_for_node_1_to_120_seconds">5. Set the expiry delay for node 1 to 120 seconds</h3>
<div class="section">
<markup
lang="bash"

>cohctl set cache test -a expiryDelay -v 120 -s PartitionedCache -n 1</markup>

<p>Output:</p>

<markup
lang="bash"

>Using cluster connection 'local' from current context.

Selected service/cache: PartitionedCache/test
Are you sure you want to set the value of attribute expiryDelay to 120 in tier back for 1 node(s)? (y/n) y
operation completed</markup>

</div>

<h3 id="_6_re_query_the_expiry_delay_by_describing_the_cache">6. Re-query the expiry delay by describing the cache</h3>
<div class="section">
<markup
lang="bash"

>cohctl describe cache test -s PartitionedCache -o jsonpath="$.items[?(@.name == 'test')]..['expiryDelay','nodeId']" |jq</markup>

<p>Output:</p>

<markup
lang="bash"

>[
  30,
  "3",
  120,
  "1",
  30,
  "2"
]</markup>

</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://docs.oracle.com/en/middleware/standalone/coherence/14.1.1.2206/manage/oracle-coherence-mbeans-reference.html#GUID-A443DF50-F151-4E9B-AFC9-DFEDF4B149E7">Cache MBean Reference</a></p>

</li>
</ul>
</div>
</div>
</doc-view>
