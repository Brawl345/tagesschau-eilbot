{
  config,
  lib,
  pkgs,
  ...
}:

let
  cfg = config.services.tagesschau-eilbot;
  defaultUser = "tagesschaueilbot";
  inherit (lib)
    mkEnableOption
    mkMerge
    mkPackageOption
    mkOption
    mkIf
    types
    optional
    optionalAttrs
    optionalString
    ;
in
{
  imports = [
    (lib.mkRemovedOptionModule [
      "services"
      "tagesschau-eilbot"
      "adminId"
    ] "The bot does not use an admin ID.")
  ];

  options.services.tagesschau-eilbot = {
    enable = mkEnableOption "Tagesschau Breaking News bot for Telegram";

    package = mkPackageOption pkgs "tagesschau-eilbot" { };

    user = mkOption {
      type = types.str;
      default = defaultUser;
      description = "User under which Telegram Breaking News Bot runs.";
    };

    group = mkOption {
      type = types.str;
      default = defaultUser;
      description = "Group under which Telegram Breaking News Bot runs.";
    };

    botTokenFile = mkOption {
      type = types.path;
      description = "File containing Telegram Bot Token";
    };

    debug = mkOption {
      type = types.bool;
      default = false;
      description = "Enable debug mode";
    };

    database = {
      host = lib.mkOption {
        type = types.str;
        description = "Database host.";
        default = "localhost";
      };

      port = mkOption {
        type = types.port;
        default = 3306;
        description = "Database port";
      };

      name = lib.mkOption {
        type = types.str;
        description = "Database name.";
        default = defaultUser;
      };

      user = lib.mkOption {
        type = types.str;
        description = "Database username.";
        default = defaultUser;
      };

      passwordFile = lib.mkOption {
        type = types.nullOr types.path;
        default = null;
        description = "Database user password file.";
      };

      socket = mkOption {
        type = types.nullOr types.path;
        default =
          if config.services.tagesschau-eilbot.database.passwordFile == null then
            "/run/mysqld/mysqld.sock"
          else
            null;
        example = "/run/mysqld/mysqld.sock";
        description = "Path to the unix socket file to use for authentication.";
      };

      createLocally = mkOption {
        type = types.bool;
        default = true;
        description = "Create the database locally";
      };
    };

  };

  config = mkIf cfg.enable {

    assertions = [
      {
        assertion = !(cfg.database.socket != null && cfg.database.passwordFile != null);
        message = "Only one of services.tagesschau-eilbot.database.socket or services.tagesschau-eilbot.database.passwordFile can be set.";
      }
      {
        assertion = cfg.database.socket != null || cfg.database.passwordFile != null;
        message = "Either services.tagesschau-eilbot.database.socket or services.tagesschau-eilbot.database.passwordFile must be set.";
      }
    ];

    services.mysql = lib.mkIf cfg.database.createLocally {
      enable = lib.mkDefault true;
      package = lib.mkDefault pkgs.mariadb;
      ensureDatabases = [ cfg.database.name ];
      ensureUsers = [
        {
          name = cfg.database.user;
          ensurePermissions = {
            "${cfg.database.name}.*" = "ALL PRIVILEGES";
          };
        }
      ];
    };

    systemd.services.tagesschau-eilbot = {
      description = "Tagesschau Breaking News Bot for Telegram";
      after = [ "network-online.target" ] ++ optional cfg.database.createLocally "mysql.service";
      wants = [ "network-online.target" ];
      requires = optional cfg.database.createLocally "mysql.service";
      wantedBy = [ "multi-user.target" ];

      script = ''
        export BOT_TOKEN="$(< $CREDENTIALS_DIRECTORY/BOT_TOKEN )"
        ${optionalString (cfg.database.passwordFile != null) ''
          export MYSQL_PASSWORD="$(< $CREDENTIALS_DIRECTORY/MYSQL_PASSWORD )"
        ''}

        exec ${cfg.package}/bin/tagesschau-eilbot
      '';

      serviceConfig = {
        LoadCredential = [
          "BOT_TOKEN:${cfg.botTokenFile}"
        ]
        ++ optional (cfg.database.passwordFile != null) "MYSQL_PASSWORD:${cfg.database.passwordFile}";

        Restart = "always";
        RestartSec = 5;
        User = cfg.user;
        Group = cfg.group;
        TimeoutStopSec = 120;

        CapabilityBoundingSet = "";
        LockPersonality = true;
        MemoryDenyWriteExecute = true;
        NoNewPrivileges = true;
        PrivateDevices = true;
        PrivateTmp = true;
        PrivateUsers = true;
        ProcSubset = "pid";
        ProtectClock = true;
        ProtectControlGroups = true;
        ProtectHome = true;
        ProtectHostname = true;
        ProtectKernelLogs = true;
        ProtectKernelModules = true;
        ProtectKernelTunables = true;
        ProtectProc = "invisible";
        ProtectSystem = "strict";
        RemoveIPC = true;
        RestrictAddressFamilies = [
          "AF_INET"
          "AF_INET6"
          "AF_UNIX"
        ];
        RestrictNamespaces = true;
        RestrictRealtime = true;
        RestrictSUIDSGID = true;
        SystemCallArchitectures = "native";
        SystemCallFilter = [
          "@system-service"
          "~@privileged"
          "~@resources"
        ];
        UMask = "0077";
      };

      environment = mkMerge [
        {
          MYSQL_HOST = cfg.database.host;
          MYSQL_PORT = toString cfg.database.port;
          MYSQL_USER = cfg.database.user;
          MYSQL_DB = cfg.database.name;
          MYSQL_SOCKET = cfg.database.socket;
        }
        (mkIf cfg.debug {
          DEBUG = "true";
        })
      ];
    };

    users.users = optionalAttrs (cfg.user == defaultUser) {
      ${defaultUser} = {
        isSystemUser = true;
        group = cfg.group;
        description = "Tagesschau Breaking News Bot user";
      };
    };

    users.groups = optionalAttrs (cfg.group == defaultUser) {
      ${defaultUser} = { };
    };

  };

}
