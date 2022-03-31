<doc-view>

<h2 id="_global_flags">Global Flags</h2>
<div class="section">
<p>The Coherence CLI provides a number of global flags that are available in all the commands. These flags are described below.</p>

<markup
lang="bash"

>Flags:
      --config string       config file (default is $HOME/.cohctl/cohctl.yaml)
      --config-dir string   config directory (default is $HOME/.cohctl)
  -c, --connection string   cluster connection name. (not required if context is set)
  -d, --delay int32         delay for watching in seconds (default 5)
  -h, --help                help for cohctl
  -o, --output string       output format: table, wide, json or jsonpath="..." (default "table")
  -i, --stdin               read password from stdin
  -U, --username string     basic auth username if authentication is required
  -w, --watch               watch output (only available for get commands)</markup>


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
