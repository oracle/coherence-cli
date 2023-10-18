<doc-view>

<h2 id="_http_servers">Http Servers</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage http servers.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-http-servers" @click.native="this.scrollFix('#get-http-servers')"><code>cohctl get http-servers</code></router-link> - displays the http servers for a cluster</p>

</li>
<li>
<p><router-link to="#describe-http-server" @click.native="this.scrollFix('#describe-http-server')"><code>cohctl describe http-server</code></router-link> - shows information related to a specific http server</p>

</li>
</ul>

<h4 id="get-http-servers">Get Http Servers</h4>
<div class="section">
<p>The 'get http-servers' command displays the list of http proxy servers for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get http-servers [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for http-servers</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display all http servers.</p>

<markup
lang="bash"

>cohctl get http-servers -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>NODE ID  HOST IP        SERVICE NAME         SERVER TYPE                  REQUESTS  ERRORS
1        0.0.0.0:0      ManagementHttpProxy  com..http.DefaultHttpServer         0       0
2        0.0.0.0:30000  ManagementHttpProxy  com..http.DefaultHttpServer        52       0
3        0.0.0.0:0      ManagementHttpProxy  com..http.DefaultHttpServer         0       0</markup>

<div class="admonition note">
<p class="admonition-inline">You can also use <code>-o wide</code> to display more columns.</p>
</div>
</div>

<h4 id="describe-http-server">Describe Http Server</h4>
<div class="section">
<p>The 'describe http-server' command shows information related to http servers.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe http-server service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for http-server</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe http-proxy ManagementHttpProxy -c local</markup>

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
