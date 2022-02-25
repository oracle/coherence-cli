<doc-view>

<h2 id="_using_jsonpath">Using JSONPath</h2>
<div class="section">
<p>JSONPath is a standard way to query elements of a JSON object. The Coherence CLI allows you to use JSONPath expressions
to filter and select data when retrieving information.</p>

<p>Below are some examples of how you could use JSONPath with the CLI. This is not an exhaustive list and the use is only limited by you imagination!</p>

<div class="admonition note">
<p class="admonition-inline">In the examples below, we are also using the <a id="" title="" target="_blank" href="https://github.com/stedolan/jq">jq</a> utility to format the JSON output.</p>
</div>
<ul class="ulist">
<li>
<p><router-link to="#cluster" @click.native="this.scrollFix('#cluster')">Cluster Examples</router-link></p>

</li>
<li>
<p><router-link to="#services" @click.native="this.scrollFix('#services')">Services Examples</router-link></p>

</li>
<li>
<p><router-link to="#members" @click.native="this.scrollFix('#members')">Members Examples</router-link></p>

</li>
<li>
<p><router-link to="#caches" @click.native="this.scrollFix('#caches')">Caches Examples</router-link></p>

</li>
<li>
<p><router-link to="#http-proxy" @click.native="this.scrollFix('#http-proxy')">Http Proxies Examples</router-link></p>

</li>
</ul>

<h3 id="cluster">Cluster</h3>
<div class="section">
<p>Retrieve the cluster name, version and cluster size from <code>cohctl describe cluster</code>.</p>

<markup
lang="bash"

>$ cohctl describe cluster my-cluster -o jsonpath="$.cluster..['clusterName', 'version', 'clusterSize']"
["cluster1","21.12",2]</markup>

</div>

<h3 id="services">Services</h3>
<div class="section">
<p>Get all service members where the requestAverageDuration &gt; 10 millis.</p>

<markup
lang="bash"

>$ cohctl get services -o jsonpath="$.items[?(@.requestAverageDuration &gt; 10)]..['nodeId','name','requestAverageDuration']"  | jq
[
  "6",
  "PartitionedTopic",
  11.815331,
  "5",
  "PartitionedTopic",
  14.489567,
  "10",
  "PartitionedTopic",
  11.648249,
  "7",
  "PartitionedCache",
  13.946078
]</markup>

</div>

<h3 id="members">Members</h3>
<div class="section">
<p>Get all members where the available memory &lt; 250MB</p>

<markup
lang="bash"

>$ cohctl get members -o jsonpath="$.items[?(@.memoryAvailableMB &lt; 250)]..['nodeId','memoryMaxMB','memoryAvailableMB']" | jq
[
  "9",
  256,
  221
]</markup>

</div>

<h3 id="caches">Caches</h3>
<div class="section">
<p>Get caches where total puts &gt; 10000.</p>

<markup
lang="bash"

>$ cohctl get caches -o jsonpath="$.items[?(@.totalPuts &gt; 10000)]..['service','name','totalPuts']" | jq
[
  "PartitionedCache2",
  "test-3",
  2000000,
  "PartitionedCache",
  "test",
  220000,
  "PartitionedCache2",
  "test-2",
  23000
]</markup>

</div>

<h3 id="http-proxy">Http Proxy Servers</h3>
<div class="section">
<p>Get http proxy servers where total request count &gt; 40.</p>

<markup
lang="bash"

>$ cohctl get http-servers -o jsonpath="$.items[?(@.totalRequestCount &gt; 15)]..['nodeId','name','totalRequestCount']"
["1","ManagementHttpProxy",45]</markup>

<p>Get persistence where latency average &gt; 0.020ms or 20 micros.</p>

<markup
lang="bash"

>$ cohctl get persistence -o jsonpath="$.items[?(@.persistenceLatencyAverage &gt; 0.020)]..['nodeId','name','persistenceLatencyAverage']" | jq
[
  "4",
  "PartitionedCache2",
  0.027694767,
  "3",
  "PartitionedCache2",
  0.029615732,
  "2",
  "PartitionedCache2",
  0.027542727,
  "1",
  "PartitionedCache2",
  0.02668317
]</markup>

</div>
</div>

<h2 id="_see_also">See Also</h2>
<div class="section">
<ul class="ulist">
<li>
<p><a id="" title="" target="_blank" href="https://earthly.dev/blog/jq-select/">An Introduction to JQ</a></p>

</li>
<li>
<p><a id="" title="" target="_blank" href="https://goessner.net/articles/JsonPath/">JSONPath examples</a></p>

</li>
<li>
<p><router-link to="/docs/about/03_quickstart">Run the Quick Start</router-link></p>

</li>
</ul>
</div>
</doc-view>
