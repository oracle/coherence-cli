<doc-view>

<h2 id="_contexts">Contexts</h2>
<div class="section">

<h3 id="_overview">Overview</h3>
<div class="section">
<p>A context allows you to specify which cluster connection you are working with, so you do no have to specify the
connection option, <code>-c</code>, with each command.</p>

<p>These include:</p>

<ul class="ulist">
<li>
<p><router-link to="#set-context" @click.native="this.scrollFix('#set-context')"><code>cohctl set context</code></router-link> - Sets the context</p>

</li>
<li>
<p><router-link to="#get-context" @click.native="this.scrollFix('#get-context')"><code>cohctl get context</code></router-link> - Get the current context</p>

</li>
<li>
<p><router-link to="#clear-context" @click.native="this.scrollFix('#clear-context')"><code>cohctl clear context</code></router-link> - Clears the current context</p>

</li>
</ul>


<h4 id="set-context">Set Context</h4>
<div class="section">
<p>The 'set context' command sets the current context or connection for running commands in.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set context connection-name [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for context</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl set context local</markup>

<p>Output:</p>

<markup
lang="bash"

>Current context is now local</markup>

</div>


<h4 id="get-context">Get Context</h4>
<div class="section">
<p>The 'get context' command displays the current context.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get context [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for context</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl get context</markup>

<p>Output:</p>

<markup
lang="bash"

>Current context: local</markup>

</div>


<h4 id="clear-context">Clear Context</h4>
<div class="section">
<p>The 'clear context' command clears the current context for running commands in.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl clear context [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for context</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>cohctl clear context</markup>

<p>Output:</p>

<markup
lang="bash"

>Current context was cleared</markup>

</div>

</div>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/reference/clusters">Clusters</router-link></p>

</li>
</ul>

</div>

</div>

</doc-view>
