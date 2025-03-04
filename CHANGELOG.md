 #  AdGuard DNS Changelog

All notable environment, configuration file, and other changes to this project
will be documented in this file.

The format is **not** based on [Keep a Changelog][kec], since the project
**doesn't** currently adhere to [Semantic Versioning][sem].

[kec]: https://keepachangelog.com/en/1.0.0/
[sem]: https://semver.org/spec/v2.0.0.html



##  AGDNS-986 / Build 346

 *  The new object `upstream.healthcheck` now contains all healthcheck-related
    fields, including the new field `domain_template`.  Property
    `upstream.healthcheck_backoff_time` has been moved to
    `upstream.healthcheck.backoff_duration`.  So replace this:

    ```yaml
    upstream:
        server: 127.0.0.1:53
        timeout: 2s
        healthcheck_enabled: true
        healthcheck_interval: 2s
        healthcheck_timeout: 1s
        healthcheck_backoff_time: 30s
        fallback:
          - 1.1.1.1:53
          - 8.8.8.8:53
    ```

    with this:

    ```yaml
    upstream:
        server: 127.0.0.1:53
        timeout: 2s
        fallback:
          - 1.1.1.1:53
          - 8.8.8.8:53
        healthcheck:
            enabled: true
            interval: 2s
            timeout: 1s
            backoff_duration: 30s
            domain_template: '${RANDOM}.neverssl.com'
    ```

    Adjust the new value, if necessary.



##  AGDNS-960 / Build 342

 *  The property `domain` of `check` object has been changed to `domains`.
    So replace this:

    ```yaml
    check:
       domain: "example.com"
    ```

    with this:

    ```yaml
    check:
       domains:
       - 'example.com'
       - 'example.org'
    ```

    Adjust the news values, if necessary.



##  AGDNS-838 / Build 338

 *  The object `upstream` has new properties, `healthcheck_enabled`,
    `healthcheck_interval`, `healthcheck_timeout`, and
    `healthcheck_backoff_time`.
    So replace this:

    ```yaml
    upstream:
        server: 127.0.0.9:53
        timeout: 2s
        fallback:
        - 1.1.1.1:53
        - 8.8.8.8:53
    ```

    with this:

    ```yaml
    upstream:
        server: 127.0.0.9:53
        timeout: 2s
        healthcheck_enabled: true
        healthcheck_interval: 2s
        healthcheck_timeout: 1s
        healthcheck_backoff_time: 30s
        fallback:
        - 1.1.1.1:53
        - 8.8.8.8:53
    ```

    Adjust the new values, if necessary.



##  Build 336

 *  The environment variable `SSLKEYLOGFILE` has been renamed to
    `SSL_KEY_LOG_FILE`.



##  AGDNS-915 / Build 334

 *  The properties `subnet_key_ip_4_mask_len` and `subnet_key_ip_6_mask_len` of
    object `ratelimit` have been renamed to `ipv4_subnet_key_len` and
    `ipv6_subnet_key_len` correspondingly.  So replace this:

    ```yaml
    ratelimit:
        # …
        subnet_key_ip_4_mask_len: 24
        subnet_key_ip_6_mask_len: 48
    ```

    with this:

    ```yaml
    ratelimit:
        # …
        ipv4_subnet_key_len: 24
        ipv6_subnet_key_len: 48
    ```



##  AGDNS-915 / Build 333

 *  The `ratelimit` object has two new properties, `subnet_key_ip_4_mask_len`
    and `subnet_key_ip_6_mask_len`.  So replace this:

    ```yaml
    ratelimit:
        # …
    ```

    with this:

    ```yaml
    ratelimit:
        # …
        subnet_key_ip_4_mask_len: 24
        subnet_key_ip_6_mask_len: 48
    ```



##  AGDNS-897 / Build 329

 *  The objects within the `filtering_groups` have a new property,
    `block_private_relay`.

    ```yaml
    filtering_groups:
    -
        id: default
        # …
    ```

    with this:

    ```yaml
    filtering_groups:
    -
        id: default
        # …
        block_private_relay: false
    ```

    The recommended default value is `false`.



##  AGDNS-624 / Build 320

 *  The objects within `server_groups` array had a change in their DDR
    configuration.  There was an opinion that the previous configuration was too
    limiting and that denormalized configuration is more self-describing. So
    replace this:

    ```yaml
    server_groups:
    -
        # …
        ddr_names:
        - 'dns.example.com'
        # …
    ```

    with this:

    ```yaml
    server_groups:
    -
        # …
        ddr:
            enabled: true
            device_records:
                '*.d.dns.example.com':
                    doh_path: '/dns-query{?dns}'
                    https_port: 443
                    quic_port: 853
                    tls_port: 853
                    ipv4_hints:
                    - 127.0.0.1
                    ipv6_hints:
                    - '::1'
            public_records:
                'dns.example.com':
                    doh_path: '/dns-query{?dns}'
                    https_port: 443
                    quic_port: 853
                    tls_port: 853
                    ipv4_hints:
                    - 127.0.0.1
                    ipv6_hints:
                    - '::1'
        # …
    ```

    Adjust the values, if necessary.  Make sure to synchronize and keep in sync
    the addresses and ports with the values of the server groups' servers.



##  AGDNS-624 / Build 317

 *  The objects within `server_groups` array have a new property `ddr_names`:

    ```yaml
    server_groups:
    -
        # …
        ddr_names:
        - 'dns.example.com'
        # …
    ```

    It is empty by default.  These values will be used for constructing a
    response for Discovery of Designated Resolvers.  Empty value leads to a
    NODATA response.  Adjust the new value, if necessary.



##  AGDNS-624 / Build 314

 *  The property `tls` of objects within the `server_groups.*.servers.*` array
    has been moved to the `server_group` object becoming common for the whole
    group.  Any group having at least a single server of DoH/DoT/DoQ protocols
    will require the `tls` property specified.  Any group having no encrypted
    resolvers will require the `tls` property absence.  So replace this:

    ```yaml
    server_groups:
    -
        # …
        servers:
        -
            name: default_dot
            protocol: tls
            tls:
                # …
            # …
    ```

    with this:

    ```yaml
    server_groups:
    -
        tls:
            # …
        # …
        servers:
        -
            name: default_dot
            protocol: tls
            # …
    ```

    Adjust the new value, if necessary.



##  AGDNS-829 / Build 308

 *  The object `upstream` has a new property, `timeout`.  So replace this:

    ```yaml
    upstream:
        server: 127.0.0.9:53
        fallback:
        - 1.1.1.1:53
        - 8.8.8.8:53
    ```

    with this:

    ```yaml
    upstream:
        server: 127.0.0.9:53
        timeout: 2s
        fallback:
        - 1.1.1.1:53
        - 8.8.8.8:53
    ```

    Adjust the new value, if necessary.



##  AGDNS-286 / Build 307

 *  The new object `connectivity_check` has been added:

    ```yaml
    connectivity_check:
        probe_ipv4: '8.8.8.8:53'
        probe_ipv6: '[2001:4860:4860::8888]:53'
    ```



##  AGDNS-745 / Build 298

 *  The object `filters` has a new property, `refresh_timeout`.  So replace
    this:

    ```yaml
    filters:
        response_ttl: 5m
        custom_filter_cache_size: 1024
        refresh_interval: 1h
    ```

    with this:

    ```yaml
    filters:
        response_ttl: 5m
        custom_filter_cache_size: 1024
        refresh_interval: 1h
        refresh_timeout: 5m
    ```

    Adjust the values, if necessary.



##  AGDNS-608 / Build 273

 *  The object `cache` has two new properties, `type` and `ecs_size`.  So
    replace this:

    ```yaml
    cache:
        size: 10000
    ```

    with this:

    ```yaml
    cache:
        type: "simple"
        size: 10000
        ecs_size: 10000
    ```

    Adjust the values, if necessary.



##  AGDNS-327 / Build 259

 *  Prometheus metric `dns_tls_handshake_total` has been updated with
    `server_name` label.  This label represents "Server Name Indication"
    identifiers, grouped by endpoint identifier and known server names.  All
    unknown server names are grouped in `other` label:

    ```
    # TYPE dns_tls_handshake_total counter
    dns_tls_handshake_total{cipher_suite="TLS_AES_128_GCM_SHA256",did_resume="0",negotiated_proto="",proto="tls",server_name="default_dot: other",tls_version="tls1.3"} 4
    ```



##  AGDNS-607 / Build 258

 *  The special "disallow-all" response is served on `/robots.txt` requests to
    `web` module.



##  AGDNS-506 / Build 242

 *  The property `cache_size` of object `geoip` has been renamed to
    `ip_cache_size`.  Also, a new property named `host_cache_size` has been
    added.  So replace this:

    ```yaml
    geoip:
        cache_size: 100000
        refresh_interval: 1h
    ```

    with this:

    ```yaml
    geoip:
        host_cache_size: 100000
        ip_cache_size: 100000
        refresh_interval: 1h
    ```

    Adjust the new value, if necessary.



##  AGDNS-505 / Build 238

 *  The object `backend` has a new property, `bill_stat_interval`.  So replace
    this:

    ```yaml
    backend:
        timeout: 10s
        refresh_interval: 15s
        full_refresh_interval: 24h
    ```

    with this:

    ```yaml
    backend:
        timeout: 10s
        refresh_interval: 15s
        full_refresh_interval: 24h
        bill_stat_interval: 15s
    ```

    Adjust the value, if necessary.



##  AGDNS-187 / Build 228

 *  The new required environment variables `GENERAL_SAFE_SEARCH_URL` and
    `YOUTUBE_SAFE_SEARCH_URL` has been added.  Those are expected to lead to
    plain text filters, for example:

    ```sh
    GENERAL_SAFE_SEARCH_URL='https://adguardteam.github.io/HostlistsRegistry/assets/engines_safe_search.txt'
    YOUTUBE_SAFE_SEARCH_URL='https://adguardteam.github.io/HostlistsRegistry/assets/youtube_safe_search.txt'
    ```



##  AGDNS-344 / Build 226

 *  The environment variables `CONSUL_DNSCHECK_KV_URL` and
    `CONSUL_DNSCHECK_SESSION_URL` are now unset by default.  Which means that by
    default HTTP key-value database isn't used.



##  AGDNS-431 / Build 211

 *  The object `web` has a new optional property, `linked_ip`:

    ```yaml
    web:
        linked_ip:
            bind:
            -
                address: 127.0.0.1:80
            -
                address: 127.0.0.1:443
                certificates:
                -
                    certificate: ./test/cert.crt
                    key: ./test/cert.key
    ```



##  AGDNS-425 / Build 209

 *  The objects within the `server_groups.*.servers` array have a new optional
    property, `linked_ip_enabled`.  It is `false` by default.  Set to `true` to
    enable linked IP address detection on that server:

    ```yaml
    server_groups:
    -
        # …
        servers:
        -
            name: default_dns
            protocol: dns
            linked_ip_enabled: true
            # …
    ```



##  AGDNS-405 / Build 195

 *  Used our fork of miekg/dns library to fix the EDNS0 TCP keep-alive issue.



##  AGDNS-341 / Build 183

 *  Removed the static DNS check `/info.txt`.  Now that `web` module is
    available, it is no more needed since it can be configured via the `web`
    module.



##  AGDNS-341 / Build 179

 *  The object `doh` has been removed.

 *  The new optional object `web` has been added:

    ```yaml
    web:
        safe_browsing:
            bind:
            -
                address: 127.0.0.1:80
            -
                address: 127.0.0.1:443
                certificates:
                -
                    certificate: ./test/cert.crt
                    key: ./test/cert.key
            block_page: /path/to/block_page.html
        adult_blocking:
            bind:
            -
                address: 127.0.0.1:80
            -
                address: 127.0.0.1:443
                certificates:
                -
                    certificate: ./test/cert.crt
                    key: ./test/cert.key
            block_page: /path/to/block_page.html
        non_doh_bind:
        -
            address: 127.0.0.1:80
        -
            address: 127.0.0.1:443
            certificates:
            -
                certificate: ./test/cert.crt
                key: ./test/cert.key
        static_content:
            '/favicon.ico':
                content_type: image/x-icon
                content: base64content
        root_redirect_url: "https://adguard-dns.com"
        error_404: /path/to/error_404.html
        error_500: /path/to/error_500.html
        timeout: 1m
    ```



##  AGDNS-367 / Build 164

 *  The object `geoip` has a new property, `cache_size`.



##  AGDNS-310 / Build 153

 *  The environment variable `LOG_OUTPUT` has been removed.  Logs are now always
    written to stdout.



##  AGDNS-339 / Build 136

 *  The environment variable `DNSDB_PATH` is now unset by default.  Which means
    that by default DNSDB is disabled.



## AGDNS-350 / Build 135

 *  The new optional environment variable `SSLKEYLOGFILE` has been added.



##  AGDNS-345 / Build 133

 *  The object `check` has a new property, `node_location`.



##  AGDNS-322 / Build 116

 *  The property `device_id_wildcard_domains` in the objects within the
    `server_groups.*.servers` array has been renamed to the shorter
    `device_id_wildcards`.

 *  The DNS names from certificates are not used to detect device IDs and
    perform additional validations anymore.



##  AGDNS-305 / Build 114

 *  The new required environment variable `BLOCKED_SERVICE_INDEX_URL` has been
    added.  It has no default value, so it's necessary to set it.



##  AGDNS-319 / Build 113

 *  The objects within the `server_groups.*.servers` array have a new property,
    `tls.device_id_wildcard_domains`.  It is an array of domain name wildcards
    used to detect device IDs.  If necessary, add them:

    ```yaml
    server_groups:
    -
        # …
        servers:
        -
            name: default_dot
            # …
            tls:
                # …
                device_id_wildcard_domains:
                - *.dns.adguard.com
    ```



##  AGDNS-292 / Build 111

 *  The environment variable `CONSUL_URL` has been renamed to
    `CONSUL_ALLOWLIST_URL`.

 *  The new required environment variables `CONSUL_DNSCHECK_KV_URL` and
    `CONSUL_DNSCHECK_SESSION_URL` are added.  They have no default value, so
    it's necessary to set them.

 *  The object `check` has a new property, `ttl`.  Set it to a human-readable
    duration, for example `1m`.



##  AGDNS-296 / Build 110

 *  The property `parental.safe_search` of objects within the `filtering_groups`
    array is renamed to `parental.general_safe_search` to synchronize it with
    the backend.



##  Build 109

 *  The object `log` has been removed.  Its properties have been moved to the
    environment.

 *  The new environment variable `LOG_OUTPUT` has been added.  It is the path to
    the plain text log file.  If `stdout`, writes to standard output.  If
    `stderr`, writes to standard error.

    The default value is `stdout`, adjust the value, if necessary.

 *  The new environment variable `LOG_TIMESTAMP` has been added.  When it is set
    to `1`, timestamps are shown in the plain text logs.  When set to `0`, they
    are
    not shown.

    The default value is `1`, adjust the value, if necessary.

 *  The environment variable `VERBOSE` doesn't support a set but empty value.
    Unset the value or replace it with a `0`.



##  AGDNS-295 / Build 105

 *  Another change in the objects within the `filtering_groups`.  Before:

    ```yaml
    filtering_groups:
    -
        id: default
        filters:
        - adguard_dns_filter
        parental: true
        block_adult: true
        safe_browsing: true
        safe_search: true
        youtube_safe_search: true
    ```

    After:

    ```yaml
    filtering_groups:
    -
        id: default
        parental:
            enabled: true
            block_adult: true
            safe_search: true
            youtube_safe_search: true
        rule_lists:
            enabled: true
            ids:
            - adguard_dns_filter
        safe_browsing:
            enabled: true
    ```



##  AGDNS-290 / Build 97

 *  The object `check` has a new property, `node_name`.



##  AGDNS-287 / Build 96

 *  The objects within the `server_groups.*.servers` array have a new optional
    property in their `dnscrypt` objects, `inline`.  Also, the property `config`
    is renamed to `config_path`.  So replace this:

    ```yaml
    server_groups:
    -
        name: adguard_dns_default
        filtering_group: default
        servers:
        -
            name: default_dnscrypt
            # …
            dnscrypt:
                config: './test/dnscrypt.yml'
        # …
    ```

    with this:

    ```yaml
    server_groups:
    -
        name: adguard_dns_default
        filtering_group: default
        servers:
        -
            name: default_dnscrypt
            # …
            dnscrypt:
                inline:
                    provider_name: 2.dnscrypt-cert.example.org
                    public_key: F11DDBCC4817E543845FDDD4CB881849B64226F3DE397625669D87B919BC4FB0
                    private_key: 5752095FFA56D963569951AFE70FE1690F378D13D8AD6F8054DFAA100907F8B6F11DDBCC4817E543845FDDD4CB881849B64226F3DE397625669D87B919BC4FB0
                    resolver_secret: 9E46E79FEB3AB3D45F4EB3EA957DEAF5D9639A0179F1850AFABA7E58F87C74C4
                    resolver_public: 9327C5E64783E19C339BD6B680A56DB85521CC6E4E0CA5DF5274E2D3CE026C6B
                    es_version: 1
                    certificate_ttl: 8760h
        # …
    ```

    or this:

    ```yaml
    server_groups:
    -
        name: adguard_dns_default
        filtering_group: default
        servers:
        -
            name: default_dnscrypt
            # …
            dnscrypt:
                config_path: './test/dnscrypt.yml'
        # …
    ```

    Adjust the values, if necessary.



##  AGDNS-290 / Build 95

 *  The property `server_name` of object `check` is removed.



##  AGDNS-272 / Build 94

 *  The new optional object `doh` has been added, which supplements the
    DNS-over-HTTP server configuration.  Example:

    ```yaml
    doh:
        root_redirect_url: "https://adguard-dns.com/"
    ```



##  AGDNS-140 / Build 90

 *  The objects within the `server_groups.*.servers` array have a new property,
    `tls.session_keys`.  So, if necessary, replace this:

    ```yaml
    server_groups:
    -
        name: adguard_dns_default
        filtering_group: default
        servers:
        -
            name: default_dot
            # …
            tls:
                certificates:
                -
                    certificate: ./test/cert.crt
                    key: ./test/cert.key
        # …
    ```

    with this:

    ```yaml
    server_groups:
    -
        name: adguard_dns_default
        filtering_group: default
        servers:
        -
            name: default_dot
            # …
            tls:
                certificates:
                -
                    certificate: ./test/cert.crt
                    key: ./test/cert.key
                session_keys:
                - ./private/key_1
        # …
    ```



##  AGDNS-233 / Build 88

 *  The object `backend` has a new property, `full_refresh_interval`.  So
    replace this:

    ```yaml
    backend:
        timeout: 10s
        refresh_interval: 1m
    ```

    with this:

    ```yaml
    backend:
        timeout: 10s
        refresh_interval: 1m
        full_refresh_interval: 24h
    ```

    Adjust the value, if necessary.



##  AGDNS-247 / Build 86

 *  The new object `check` has been added, which configures the DNS checks
    mechanism.  Example:

    ```yaml
    check:
        domain: "dnscheck.adguard.com"
        ipv4:
        - 1.2.3.4
        - 5.6.7.8
        ipv6:
        - 1234::cdee
        - 1234::cdef
        server_name: "AdGuard DNS Default"
    ```



##  AGDNS-246 / Build 83

 *  The new environment variable `RULESTAT_URL` has been added.  Its default
    value is <code></code>, which means that no statistics are gathered.  Adjust
    the value, if necessary.



##  AGDNS-245 / Build 74

 *  The new environment variable `DNSDB_PATH` has been added.  Its default value
    is `./dnsdb.bolt`.  Adjust the value, if necessary.



##  AGDNS-139 / Build 73

 *  The new required environment variable `CONSUL_URL` has been added.  It has
    no default value, so it's necessary to set it.

 *  The ratelimit configuration for a server has changed from this:

    ```yaml
    ratelimit:
        refuseany: true
        response_size_limit: 1KB
        rate_limit_cache_ttl: 10m
        back_off_cache_ttl: 30m
        rps: 30
        backoff_limit: 1000
    ```

    to this:

    ```yaml
    ratelimit:
        allowlist:
            list:
            - '127.0.0.1'
            - '127.0.0.1/24'
            refresh_interval: 30s
        back_off_count: 1000
        back_off_duration: 30m
        back_off_period: 10m
        refuseany: true
        response_size_estimate: 1KB
        rps: 30
    ```

    See README.md for documentation.



##  AGDNS-154 / Build 71

 *  The property `backend` of the `query_log` object is removed.



##  AGDNS-230 / Build 67

 *  The new required environment variable `FILTER_INDEX_URL` has been added.  It
    has no default value, so it's necessary to set it.

 *  The environment variable `BACKEND_ENDPOINT` is now required and has no
    default value.

 *  Property `lists` of the `filters` object is removed.

 *  A new property `refresh_interval` has been added to the `filters` object.



##  AGDNS-229 / Build 62

 *  The new environment variable `FILTER_CACHE_PATH` has been added.  Its
    default value is `./filters/`.  Adjust the value, if necessary.

 *  The `list` property of `safe_browsing` and `adult_blocking` objects as well
    as the `path` property of the `filters.lists` objects are removed.

 *  Property `url` of the `filters.lists` objects is now required.



##  AGDNS-188 / Build 61

 *  The type of the `cache.size` property was changed from bytes to integer.  So
    replace this:

    ```yaml
    cache:
        size: 50KB
    ```

    with this:

    ```yaml
    cache:
        size: 10000
    ```

    Set the new values accordingly.



##  AGDNS-149, AGDNS-150, AGDNS-189 / Build 52

 *  The top-level object `parental` was renamed to `adult_blocking`.

 *  The objects `safe_browsing` and `adult_blocking` have four new properties,
    `cache_size`, `cache_ttl`, `refresh_interval`, and `url`.  So replace this:

    ```yaml
    safe_browsing:
        block_host: standard-block.dns.adguard.com
        list: ./test/safe_browsing.txt

    adult_blocking:
        block_host: family-block.dns.adguard.com
        list: ./test/parental.txt
    ```

    with this:

    ```yaml
    safe_browsing:
        url: https://static.example.com/safe_browsing.txt
        block_host: standard-block.dns.adguard.com
        cache_size: 1024
        cache_ttl: 1h
        list: ./test/safe_browsing.txt
        refresh_interval: 1h

    adult_blocking:
        url: https://static.example.com/adult_blocking.txt
        block_host: family-block.dns.adguard.com
        cache_size: 1024
        cache_ttl: 1h
        list: ./test/parental.txt
        refresh_interval: 1h
    ```

    Set the new values accordingly.

 *  The objects within the `filtering_groups` array have a new property,
    `block_adult`.  So replace this:

    ```yaml
    filtering_groups:
    -
        id: default
        filters:
        - adguard_dns_filter
        parental: false
        safe_browsing: true
        safe_search: false
        youtube_safe_search: false
    # …
    ```

    with this:

    ```yaml
    filtering_groups:
    -
        id: default
        filters:
        - adguard_dns_filter
        parental: false
        block_adult: false
        safe_browsing: true
        safe_search: false
        youtube_safe_search: false
    # …
    ```

    Set the new value accordingly.

 *  The objects within the `filters.lists` array have a new property,
    `refresh_interval`.  The property is only required when the property `url`
    is also set.  So replace this:

    ```yaml
    filters:
        # …
        lists:
        -
            id: adguard_dns_filter
            url: 'https://example.com/adguard_dns_filter.txt'
            path: ./test/filters/adguard_dns_filter.txt
        -
            id: peter_lowe_list
            path: ./test/filters/peter_lowe_list.txt
    ```

    with this:

    ```yaml
    filters:
        # …
        lists:
        -
            id: adguard_dns_filter
            url: 'https://example.com/adguard_dns_filter.txt'
            path: ./test/filters/adguard_dns_filter.txt
            refresh_interval: 1h
        -
            id: peter_lowe_list
            path: ./test/filters/peter_lowe_list.txt
    ```

    Set the new value accordingly.



##  Build 45

 *  The property `youtube_restricted` was renamed to `youtube_safe_search`.
    So replace this:

    ```yaml
    filtering_groups:
    -
        id: default
        # …
        youtube_restricted: false
    -
        id: strict
        # …
        youtube_restricted: true
    ```

    with this:

    ```yaml
    filtering_groups:
    -
        id: default
        # …
        youtube_safe_search: false
    -
        id: strict
        # …
        youtube_safe_search: true
    ```



##  AGDNS-152 / Build 43

 *  The blocked response TTL parameter has been moved and renamed.  From this:

    ```yaml
    dns:
        blocked_response_ttl: 10s
    ```

    to this:

    ```yaml
    filters:
        response_ttl: 10s
    ```

    The `dns` object has been completely removed.



##  AGDNS-177 / Build 40

 *  The TLS configuration for a server has changed from this:

    ```yaml
    tls:
        certificates:
        -
            certificate: /test/cert.crt
            key: /test/cert.key
        domains:
        - dns.adguard.com
    ```

    to this:

    ```yaml
    tls:
        certificates:
        -
            certificate: /test/cert.crt
            key: /test/cert.key
    ```

    The domains to be used in device ID detection are now expected to be
    contained in the certificate's DNS Names section of SAN.



##  AGDNS-167 / Build 39

 *  The filtering configuration has changed from this:

    ```yaml
    filters:
    -
        id: adguard_dns_filter
        path: ./tmp.dir/filter.txt
    ```

    to this:

    ```yaml
    filters:
        custom_filter_cache_size: 1024
        lists:
        -
            id: adguard_dns_filter
            path: ./tmp.dir/filter.txt
    ```
