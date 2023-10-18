<doc-view>

<h2 id="_http_sessions">Http Sessions</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>There are various commands that allow you to work with and manage Coherence*Web Http session information.</p>

<ul class="ulist">
<li>
<p><router-link to="#get-http-sessions" @click.native="this.scrollFix('#get-http-sessions')"><code>cohctl get http-sessions</code></router-link> - displays the Http session details</p>

</li>
<li>
<p><router-link to="#describe-http-session" @click.native="this.scrollFix('#describe-http-session')"><code>cohctl describe http-session</code></router-link> - shows http session information related to a specific application id</p>

</li>
</ul>
<div class="admonition note">
<p class="admonition-inline">This is a Coherence Grid Edition feature only and is not available with Community Edition.</p>
</div>

<h4 id="get-http-sessions">Get Http Sessions</h4>
<div class="section">
<p>The 'get http-sessions' command displays Coherence*Web Http session information for a cluster.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get http-sessions [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for http-sessions</pre>
</div>

<p><strong>Examples</strong></p>

<p>Display Http Session data.</p>

<markup
lang="bash"

>cohctl get http-servers -c local</markup>

<p>Output:</p>

<markup
lang="bash"

>APPLICATION  TYPE                SESSION TIMEOUT  CACHE          OVERFLOW CACHE  AVG SIZE  TOTAL REAPED  AVG DURATION  LAST REAP  SESSION UPDATES
app-1        HttpSessionManager              600  session-cache                       103             1         1,234        123                3
app-2        HttpSessionManager              600  session-cache                      1234             0             0          0                5</markup>

</div>

<h4 id="describe-http-session">Describe Http Session</h4>
<div class="section">
<p>The 'describe http-session' command shows information related to a specific Coherence*Web application.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl describe http-session application-id [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for http-session</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl describe http-session app-1 -c local</markup>

</div>
</div>
</div>
</doc-view>
