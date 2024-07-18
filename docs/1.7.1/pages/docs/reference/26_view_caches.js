<doc-view>

<h2 id="_view_caches">View Caches</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage cluster view caches.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-view-caches" @click.native="this.scrollFix('#get-view-caches')"><code>cohctl get view-caches</code></router-link> - displays the view caches for a cluster</p>

</li>
<li>
<p><router-link to="#describe-view-cache" @click.native="this.scrollFix('#describe-view-cache')"><code>cohctl describe view-cache</code></router-link> - shows information related to a specific view cache and service</p>

</li>
</ul>

<h4 id="get-view-caches">Get View Caches</h4>
<div class="section">
<p>The 'get view-caches' command displays view caches for a cluster. If no service
name is specified then all services are queried.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get view-caches [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for view-caches
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all view caches.</p>

<markup
lang="bash"

>cohctl get view-caches -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Total View Caches: 2

SERVICE                  VIEW NAME      MEMBERS
DistributedCacheService  view-cache-1         3
DistributedCacheService  view-cache-2         3</markup>

</div>

<h4 id="describe-view-cache">Describe View Cache</h4>
<div class="section">
<p>The 'describe view-cache' command displays information related to a specific view-cache.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe view-cache cache-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help             help for view-cache
  -s, --service string   Service name</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe view-cache view-cache-1 -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>Service:    DistributedCacheService
View Cache: view-cache-1

NODE ID  VIEW SIZE  RECONNECT  FILTER        TRANSFORMED  TRANSFORMER  READ ONLY
      3          5       0.0s  AlwaysFilter  false        n/a          false
      4          5       0.0s  AlwaysFilter  false        n/a          false
      5          5       0.0s  AlwaysFilter  false        n/a          false</markup>

<div class="admonition note">
<p class="admonition-inline">You may omit the service name option if the view cache name is unique.</p>
</div>
</div>
</div>

<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/25_caches">Caches</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
