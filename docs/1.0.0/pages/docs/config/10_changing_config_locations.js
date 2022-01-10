<doc-view>

<h2 id="_changing_config_locations">Changing Config Locations</h2>
<div class="section">
<p>By default, the CLI creates a directory <code>.cohctl</code> off the users home to store connection information,
log files and various other information.</p>

<p>If you wish to change this directory from the default, you can use the following for each command.</p>

<markup
lang="bash"

>$ cohctl get clusters --config-dir /u01/config</markup>

<p>You can also specify a different location for the <code>cohct.yaml</code> file that is generated.</p>

<markup
lang="bash"

>$ cohctl get clusters --config /u01/my-config.yaml.</markup>

<div class="admonition note">
<p class="admonition-inline">It is recommended to leave these values as default unless you have a good reason to change them
as you would need to specify the <code>--config</code> option on each command execution.</p>
</div>
</div>
</doc-view>
