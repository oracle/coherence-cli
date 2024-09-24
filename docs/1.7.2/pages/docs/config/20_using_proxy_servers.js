<doc-view>

<h2 id="_using_proxy_servers">Using Proxy Servers</h2>
<div class="section">
<p>There may be cases where you wish to configure a proxy server to access the HTTP endpoint for your cluster.</p>

<p>The CLI honors the following standard environment variable settings, by internally using <a id="" title="" target="_blank" href="https://pkg.go.dev/net/http#ProxyFromEnvironment">Proxy.ProxyFromEnvironment</a>, for proxy server configuration:</p>

<ul class="ulist">
<li>
<p><code>HTTP_PROXY</code> or <code>http_proxy</code></p>

</li>
<li>
<p><code>HTTPS_PROXY</code> or <code>https_proxy</code></p>

</li>
<li>
<p><code>NO_PROXY</code> or <code>no_proxy</code></p>

</li>
</ul>
</div>
</doc-view>
