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
<p><router-link to="#get-proxy-connections" @click.native="this.scrollFix('#get-proxy-connections')"><code>cohctl get proxy-connections</code></router-link> - displays proxy server connections for a specific proxy server</p>

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

<h4 id="get-proxy-connections">Get Proxy Connections</h4>
<div class="section">
<p>The 'get proxy-connections' command displays proxy server connections for a specific proxy server.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get proxy-connections service-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for proxy-connections</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get proxy-connections Proxy -c local

NODE ID    CONN MS  CONN TIME  REMOTE ADDR/PORT  BYTES SENT  BYTES REC  BACKLOG  CLIENT PROCESS
      1    236,916    03m 56s   127.0.0.1:58819        0 MB       0 MB        0           55414
      1    538,395    08m 58s   127.0.0.1:58666        0 MB       0 MB        0           54769
      2  1,177,423    19m 37s   127.0.0.1:58075        1 MB       0 MB        0           45646</markup>

<p>You can use <code>-o wide</code> to display more columns as described below.</p>

<markup
lang="bash"

>$ cohctl get proxy-connections Proxy -o wide -c local

NODE ID    CONN MS  CONN TIME  REMOTE ADDR/PORT  BYTES SENT  BYTES REC  BACKLOG  CLIENT PROCESS  CLIENT ROLE                         UUID             TIMESTAMP
      1    275,256    04m 35s   127.0.0.1:58819        0 MB       0 MB        0           55414  TangosolCoherenceDslqueryQueryPlus  0x000001...B052  2022-09-09T13:24:36.898+08:00
      1    576,736    09m 36s   127.0.0.1:58666        0 MB       0 MB        0           54769  TangosolCoherenceDslqueryQueryPlus  0x000001...B050  2022-09-09T13:19:35.418+08:00
      2  1,215,764    20m 15s   127.0.0.1:58075        1 MB       0 MB        0           45646  TangosolCoherenceDslqueryQueryPlus  0x000001...636D  2022-09-09T13:08:55.777+08:00</markup>

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

>$ cohctl describe proxy Proxy -c local

PROXY SERVICE DETAILS
---------------------
Name                                :  Proxy
Type                                :  [Proxy]
...
Transport Backlogged Connection List:  [[] []]

PROXY MEMBER DETAILS
--------------------
NODE ID  HOST IP              SERVICE NAME  CONNECTIONS  BYTES SENT  BYTES REC
1        0.0.0.0:53962.47748  Proxy                   2       1,394      2,471
2        0.0.0.0:53966.60421  Proxy                   1   1,049,157        703

PROXY CONNECTIONS
-----------------
Node Id                 :  2
Remote Address          :  127.0.0.1
...
UUID                    :  0x0000018320A6A563C0A80189867534FD64D606EA44860A00C7DBDE274D31636D

Node Id                 :  1
Remote Address          :  127.0.0.1
...
UUID                    :  0x0000018320B067FDC0A80189C594C09E90166E1F48D3806BC52F4FFE8097B050

Node Id                 :  1
Remote Address          :  127.0.0.1
...
UUID                    :  0x0000018320B501A5C0A8018931AE66ADAC4A887A2E21D5A6C51F69858097B052</markup>

<div class="admonition note">
<p class="admonition-inline">The above output has been truncated for brevity.</p>
</div>
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
