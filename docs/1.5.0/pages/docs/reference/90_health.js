<doc-view>

<h2 id="_health">Health</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>If your cluster version supports it, you can display health information using the following commands.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-health" @click.native="this.scrollFix('#get-health')"><code>cohctl get health</code></router-link> - display health information for a cluster</p>

</li>
</ul>

<h4 id="get-health">Get Health</h4>
<div class="section">
<p>The 'get health' command displays the health for members of a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get health [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help              help for health
  -n, --name string       health name (default "all")
  -s, --sub-type string   health sub-type (default "all")
  -S, --summary           if true, returns a summary across nodes</pre>
</div>

<p><strong>Examples</strong></p>

<p>Return all the health endpoint status for the cluster.</p>

<markup
lang="bash"

>$ cohctl get health -c local

NODE ID  NAME                  SUB TYPE   STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
      1  Proxy                 Service    true     true  true   true  true           ServiceModel: type=Service,name=Proxy,nodeId=1
      1  PartitionedTopic      Service    true     true  true   true  true           ServiceModel: type=Service,name=PartitionedTopic,nodeId=1
      1  PartitionedCache      Service    true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=1
      1  MetricsHttpProxy      Service    true     true  true   true  true           ServiceModel: type=Service,name=MetricsHttpProxy,nodeId=1
      1  ManagementHttpProxy   Service    true     true  true   true  true           ServiceModel: type=Service,name=ManagementHttpProxy,nodeId=1
      1  Default               Coherence  true     true  true   true  true           com.tangosol.net.Coherence$CoherenceHealth@5fa2993b
      1  $SYS:HealthHttpProxy  Service    true     true  true   true  true           ServiceModel: type=Service,name=$SYS:HealthHttpProxy,nodeId=1
      1  $SYS:Config           Service    true     true  true   true  true           ServiceModel: type=Service,name=$SYS:Config,nodeId=1
      2  Proxy                 Service    true     true  true   true  true           ServiceModel: type=Service,name=Proxy,nodeId=2
      2  PartitionedTopic      Service    true     true  true   true  true           ServiceModel: type=Service,name=PartitionedTopic,nodeId=2
      2  PartitionedCache      Service    true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=2
      2  Default               Coherence  true     true  true   true  true           com.tangosol.net.Coherence$CoherenceHealth@39f79f18
      2  $SYS:HealthHttpProxy  Service    true     true  true   true  true           ServiceModel: type=Service,name=$SYS:HealthHttpProxy,nodeId=2
      2  $SYS:Config           Service    true     true  true   true  true           ServiceModel: type=Service,name=$SYS:Config,nodeId=2</markup>

<p>Return health for a specific name of <code>PartitionedCache</code>.</p>

<markup
lang="bash"

>$ cohctl get health -c local -n PartitionedCache

NODE ID  NAME              SUB TYPE  STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
      1  PartitionedCache  Service   true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=1
      2  PartitionedCache  Service   true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=2</markup>

<p>Return health for a specific sub-type of <code>Coherence</code>.</p>

<markup
lang="bash"

>$ cohctl get health -c local -n PartitionedCache

NODE ID  NAME     SUB TYPE   STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
      1  Default  Coherence  true     true  true   true  true           com.tangosol.net.Coherence$CoherenceHealth@5fa2993b
      2  Default  Coherence  true     true  true   true  true           com.tangosol.net.Coherence$CoherenceHealth@39f79f18</markup>

<div class="admonition note">
<p class="admonition-inline">You can use <code>-o wide</code> to display additional information.</p>
</div>
<p><strong>Examples</strong></p>

<p>Return a health summary for the cluster for all health endpoints, by using the <code>-S</code> option,
to show how many members are in each state.</p>

<markup
lang="bash"

>$ cohctl get health -c local -S

NAME                  SUB TYPE   MEMBERS  STARTED  LIVE  READY  SAFE
Proxy                 Service          3        3     3      3     3
PartitionedTopic      Service          3        3     3      3   1/3
PartitionedCache      Service          3        3     3      3   1/3
MetricsHttpProxy      Service          1        1     1      1     1
ManagementHttpProxy   Service          1        1     1      1     1
Default               Coherence        3        3     3      3     3
$SYS:HealthHttpProxy  Service          3        3     3      3     3
$SYS:Config           Service          3        3     3      3     3</markup>

</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
