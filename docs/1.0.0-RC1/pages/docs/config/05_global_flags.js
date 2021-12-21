<doc-view>

<h2 id="_global_flags">Global Flags</h2>
<div class="section">
<p>The Coherence CLI provides a number of global flags that are available in all the commands. These flags are described below.</p>

<markup
lang="bash"

>Flags:
      --config string       Config file (default is $HOME/.cohctl/cohctl.yaml)
      --config-dir string   Config directory (default is $HOME/.cohctl)
  -c, --connection string   Cluster connection name. (not required if context is set)
  -d, --delay int32         Delay for watching in seconds (default 5)
  -h, --help                help for cohctl
  -o, --output string       Output format: table, wide, json or jsonpath="..." (default "table")
  -i, --stdin               Read password from stdin
  -U, --username string     Basic auth username if authentication is required
  -w, --watch               Watch output (only available for get commands)</markup>


<h3 id="_see_also">See Also</h3>
<div class="section">
<ul class="ulist">
<li>
<p><router-link to="/docs/config/10_changing_config_locations">Changing Config Locations</router-link></p>

</li>
<li>
<p><router-link to="/docs/examples/10_jsonpath">Using JsonPath</router-link></p>

</li>
</ul>
</div>
</div>
</doc-view>
