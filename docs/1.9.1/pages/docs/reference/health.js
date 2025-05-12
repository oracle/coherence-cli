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
<li>
<p><router-link to="#monitor-health" @click.native="this.scrollFix('#monitor-health')"><code>cohctl monitor health</code></router-link> - monitor health information for a cluster or set of health endpoints</p>

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

>cohctl get health -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  NAME                  SUB TYPE   STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
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

>cohctl get health -c local -n PartitionedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  NAME              SUB TYPE  STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
      1  PartitionedCache  Service   true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=1
      2  PartitionedCache  Service   true     true  true   true  true           ServiceModel: type=Service,name=PartitionedCache,nodeId=2</markup>

<p>Return health for a specific sub-type of <code>Coherence</code>.</p>

<markup
lang="bash"

>cohctl get health -c local -n PartitionedCache</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  NAME     SUB TYPE   STARTED  LIVE  READY  SAFE  MEMBER HEALTH  DESCRIPTION
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

>cohctl get health -c local -S</markup>

<p>Output:</p>

<markup
lang="bash"

>NAME                  SUB TYPE   MEMBERS  STARTED  LIVE  READY  SAFE
Proxy                 Service          3        3     3      3     3
PartitionedTopic      Service          3        3     3      3   1/3
PartitionedCache      Service          3        3     3      3   1/3
MetricsHttpProxy      Service          1        1     1      1     1
ManagementHttpProxy   Service          1        1     1      1     1
Default               Coherence        3        3     3      3     3
$SYS:HealthHttpProxy  Service          3        3     3      3     3
$SYS:Config           Service          3        3     3      3     3</markup>

</div>


<h4 id="monitor-health">Monitor Health</h4>
<div class="section">
<p>The 'get monitor' command monitors the health of nodes for a cluster or set of health endpoints.
Specify -n and a host:port to lookup or -e and a list of http endpoints without the path.
You may also specify -T option to wait until all health endpoints are safe.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl monitor health [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -e, --endpoints string       csv list of health endpoints
  -T, --health-timeout int32   timeout to wait for all health checks to be status 200
  -h, --help                   help for health
  -I, --ignore-errors          if true, ignores nslookup errors
  -N, --node-id                if true, returns the node id using the current context
  -n, --nslookup string        host:port to connect to to lookup health endpoints
  -t, --timeout int32          timeout in seconds for NS Lookup requests (default 30)</pre>
</div>

<div class="admonition note">
<p class="admonition-inline">This is useful especially when you want to check the health of
members during a rolling restart. Values returned are the HTTP Status codes of <code>200</code> if the health is OK, <code>503</code>
if it is not and <code>Refused</code> if the endpoint was not able to be reached.</p>
</div>

<p>See the <a target="_blank" href="https://docs.oracle.com/en/middleware/fusion-middleware/coherence/14.1.2/manage/using-health-check-api.html">Coherence Documentation</a> for more information on the health check API.</p>

<p><strong>Examples</strong></p>

<p>Monitor the health endpoints for a cluster using the name service to look up the health endpoints.</p>

<markup
lang="bash"

>cohctl monitor health -c local -n localhost:7574</markup>

<p>Output:</p>

<markup
lang="bash"

>HEALTH MONITORING
------------------
Name Service:    localhost:7574
Cluster Name:    local
All Nodes Safe:  true

URL                      NODE ID  STARTED  LIVE  READY  SAFE  OVERALL
http://127.0.0.1:63765/      n/a      200   200    200   200        4
http://127.0.0.1:63766/      n/a      200   200    200   200        4
http://127.0.0.1:63768/      n/a      200   200    200   200        4</markup>

<div class="admonition note">
<p class="admonition-inline">All Nodes Safe will indicate if all the health checks all returned HTTP 200. If you are monitoring a rolling
restart, this is an indication that it is safe to continue.</p>
</div>

<p>Use the <code>-N</code> option to display the node id from the current context.</p>

<markup
lang="bash"

>cohctl monitor health -c local -n localhost:7574 -N</markup>

<p>Output:</p>

<markup
lang="bash"

>HEALTH MONITORING
------------------
Name Service:    localhost:7574
Cluster Name:    local
All Nodes Safe:  true


URL                      NODE ID  STARTED  LIVE  READY  SAFE  OVERALL
http://127.0.0.1:63765/        2      200   200    200   200        4
http://127.0.0.1:63766/        1      200   200    200   200        4
http://127.0.0.1:63768/        3      200   200    200   200        4</markup>

<p>Monitor the health endpoints for a cluster a list of health endpoints.</p>

<div class="admonition note">
<p class="admonition-inline">The endpoints should not include any path information.</p>
</div>

<markup
lang="bash"

>cohctl monitor health -c local -e http://127.0.0.1:63768/,http://127.0.0.1:63744/,http://127.0.0.1:6544/</markup>

<p>Output:</p>

<markup
lang="bash"

>HEALTH MONITORING
------------------
Endpoints:  [http://127.0.0.1:63768/ http://127.0.0.1:63744/ http://127.0.0.1:6544/]

URL                      NODE ID  STARTED     LIVE    READY     SAFE  OVERALL
http://127.0.0.1:63744/      n/a  Refused  Refused  Refused  Refused      0/4
http://127.0.0.1:63768/      n/a      200      200      200      200        4
http://127.0.0.1:6544/       n/a  Refused  Refused  Refused  Refused      0/4</markup>

<p>Monitor the health endpoints via the name service and wait until they are all safe.</p>

<p>This is useful for scripting during a rolling restart when you want to wait until all members are safe
before proceeding.</p>

<p>You must specify the following options:</p>

<ul class="ulist">
<li>
<p><code>-T</code> specifies the number of seconds to wait until all health endpoints are safe</p>

</li>
<li>
<p><code>-w</code> or <code>-W</code> to wait</p>

</li>
</ul>

<p>If the endpoints are all safe within the time specified, <code>cohctl</code> will return 0, otherwise it will return 1.</p>

<markup
lang="bash"

>cohctl monitor health -c local -n localhost:7574 -N -T 60 -W</markup>

<p>Output:</p>

<markup
lang="bash"

>HEALTH MONITORING
------------------
Name Service:    localhost:7574
Cluster Name:    local
All Nodes Safe:  true

URL                      NODE ID  STARTED  LIVE  READY  SAFE  OVERALL
http://127.0.0.1:52133/        1      200   200    200   200        4
http://127.0.0.1:52134/        3      200   200    200   200        4
http://127.0.0.1:52136/        2      200   200    200   200        4

All health endpoints are safe reached in 21 seconds</markup>

<div class="admonition note">
<p class="admonition-inline">You can add <code>-I</code> option when using the <code>-n</code> name service option, to ignore errors connecting to the name service.</p>
</div>

</div>

</div>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/services">Services</router-link></p>

</li>
</ul>

</div>

</div>

</doc-view>
