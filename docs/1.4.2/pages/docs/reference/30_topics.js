<doc-view>

<h2 id="_topics">Topics</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage cluster topics.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-topics" @click.native="this.scrollFix('#get-topics')"><code>cohctl get topics</code></router-link> - displays the topics for a cluster</p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">These topics commands are experimental only and may change or be removed in the future.</p>
</div>

<h4 id="get-topics">Get Topics</h4>
<div class="section">
<p>The 'get topics' command displays topics for a cluster. If
no service name is specified then all services are queried.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get topics [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for topics
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all topics.</p>

<markup
lang="bash"

>$ cohctl get topics -c local
Total Topics: 3, Total primary storage: 0

SERVICE           TOPIC             UNCONSUMED MSG   SIZE  AVG SIZE  PUBLISHER SENDS  SUBSCRIBER RECEIVES
PartitionedTopic  private-messages               2   0 MB       512                2                    4
PartitionedTopic  public-messages               14   0 MB       510               18                   36
PartitionedTopic2 public-messages2              15   0 MB       510               13                   36</markup>

<div class="admonition note">
<p class="admonition-inline">If you want to describe a topic, please use <code>get caches</code> to list the topics caches
and describe the cache. See <router-link to="/docs/reference/25_caches">Caches</router-link> for more information.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/25_caches">Caches</router-link></p>

</li>
<li>
<p><router-link to="/docs/reference/20_services">Services</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
