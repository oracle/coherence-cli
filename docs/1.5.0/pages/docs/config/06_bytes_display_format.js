<doc-view>

<h2 id="_bytes_display_format">Bytes Display Format</h2>
<div class="section">
<p>By default, any memory or disk based size value is displayed in bytes. You can use the following options on any command to change the display format:</p>

<ul class="ulist">
<li>
<p><code>-k</code> or <code>--kb</code> - display in kilobytes (KB)</p>

</li>
<li>
<p><code>-m</code> or <code>--mb</code> - display in megabytes (MB)</p>

</li>
<li>
<p><code>-g</code> or <code>--gb</code> - display in gigabytes (GB)</p>

</li>
<li>
<p><code>--tb</code> - display in terabytes (TB)</p>

</li>
</ul>
<p>For the purposes of display, units of 1024 are used to calculate the appropriate value. E.g. 1 KB = 1024 bytes.</p>

<div class="admonition note">
<p class="admonition-inline">Specifying the above options will override any default you have set below.</p>
</div>

<h3 id="_setting_the_default_bytes_display_format">Setting the Default Bytes Display Format</h3>
<div class="section">
<p>If you prefer to always use a particular display format for output, you can use the following commands to control
the default format:</p>

<ul class="ulist">
<li>
<p><router-link to="#set-bytes-format" @click.native="this.scrollFix('#set-bytes-format')"><code>cohctl set bytes-format</code></router-link> - set the default bytes format</p>

</li>
<li>
<p><router-link to="#get-bytes-format" @click.native="this.scrollFix('#get-bytes-format')"><code>cohctl get bytes-format</code></router-link> - display the current bytes format</p>

</li>
<li>
<p><router-link to="#clear-bytes-format" @click.native="this.scrollFix('#clear-bytes-format')"><code>cohctl clear bytes-format</code></router-link> - clear the current bytes format</p>

</li>
</ul>

<h4 id="set-bytes-format">Set Default Bytes Format</h4>
<div class="section">
<p>The 'set bytes-format' command sets the default format for displaying memory or disk based sizes.
Valid values are k - kilobytes, m - megabytes, g - gigabytes or t - terabytes. If not specified the default will be b - bytes.
The default value will be overridden if you specify the -k, -m, -g or --tb options.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl set bytes-format {k|m|g|t} [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for bytes-format</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl set bytes-format m
Bytes format is now set to m</markup>

</div>

<h4 id="get-bytes-format">Get Default Bytes Format</h4>
<div class="section">
<p>The 'get bytes-format' displays the current format for displaying memory or disk based sizes.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl get bytes-format [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for bytes-format</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl get bytes-format
Current bytes format: m</markup>

</div>

<h4 id="clear-bytes-format">Clear Default Bytes Format</h4>
<div class="section">
<p>The 'clear bytes-format' clears the current format for displaying memory or disk based sizes.</p>

<p><strong>Usage</strong></p>

<div class="listing">
<pre>  cohctl clear bytes-format [flags]</pre>
</div>

<p><strong>Flags</strong></p>

<div class="listing">
<pre>  -h, --help   help for bytes-format</pre>
</div>

<p><strong>Examples</strong></p>

<markup
lang="bash"

>$ cohctl clear bytes-format
Default bytes format has been cleared</markup>

</div>
</div>
</div>
</doc-view>
