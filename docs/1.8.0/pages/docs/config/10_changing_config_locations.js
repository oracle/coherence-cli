<doc-view>

<h2 id="_changing_config_locations">Changing Config Locations</h2>
<div class="section">
<p>By default, the CLI creates a directory <code>.cohctl</code> off the user&#8217;s home to store connection information,
log files and various other information.</p>

<div class="admonition note">
<p class="admonition-inline">It is recommended to leave these values as default unless you have a good reason to change them.
If you do want to change the location, setting the <code>COHCTL_HOME</code> environment variable is the best option
otherwise you would need to specify the <code>--config</code> option on each command execution.</p>
</div>
<p>If you wish to change this directory from the default, you can use the following options:</p>


<h3 id="_set_cohctl_home_environment_variable">Set COHCTL_HOME environment variable</h3>
<div class="section">
<p>You can set the environment variable <code>COHCTL_HOME</code> to change the location of the directory where the CLI creates it&#8217;s files.</p>

<markup
lang="bash"

>export COHCTL_HOME=/u01/config</markup>

<p>All subsequent command executions will use this directory. The directory will be created if it doesn&#8217;t exist
and will throw an error if you don&#8217;t have permissions to create it.</p>

</div>

<h3 id="_specify_the_directory_for_each_command">Specify the directory for each command</h3>
<div class="section">
<markup
lang="bash"

>cohctl get clusters --config-dir /u01/config</markup>

<p>You can also specify a different location for the <code>cohct.yaml</code> file that is generated.</p>

<markup
lang="bash"

>cohctl get clusters --config /u01/my-config.yaml</markup>

</div>
</div>
</doc-view>
