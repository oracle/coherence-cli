<doc-view>

<h2 id="_proxy_servers">Proxy Servers</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage proxy servers.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-proxies" @click.native="this.scrollFix('#get-proxies')"><code>cohctl get proxies</code></router-link> - displays the proxy servers for a cluster</p>

</li>
<li>
<p><router-link to="#describe-proxy" @click.native="this.scrollFix('#describe-proxy')"><code>cohctl describe proxy</code></router-link> - shows information related to a specific proxy server</p>

</li>
</ul>

<h4 id="get-proxies">Get Proxies</h4>
<div class="section">
<p>The 'get proxies' command displays the list of Coherence*Extend proxy
servers for a cluster. You can specify '-o wide' to display addition information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get proxies [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for proxies</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all proxy servers.</p>

<markup
lang="bash"

>$ cohctl get proxies -c local
NODE ID  HOST IP              SERVICE NAME  CONNECTIONS  BYTES SENT  BYTES REC
1        0.0.0.0:53216.41408  Proxy                   0           0          0
2        0.0.0.0:53215.47265  Proxy                   0           0          0
3        0.0.0.0:53220.42214  Proxy                   0           0          0</markup>

<div class="admonition note">
<p class="admonition-inline">You can also use <code>-o wide</code> to display more columns.</p>
</div>
</div>

<h4 id="describe-proxy">Describe Proxy</h4>
<div class="section">
<p>The 'describe proxy' command shows information related to proxy servers including
all nodes running the proxy service as well as detailed connection information.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe proxy service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for proxy</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl describe proxy Proxy -c local</markup>

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
