{ config, lib, pkgs, ... }:

let
  cfg = config.services.tagesschau-eilbot;
  defaultUser = "tagesschau-eilbot";
  inherit (lib) mkEnableOption mkMerge mkPackageOption mkOption mkIf types optionalAttrs;
in
{
  options.services.tagesschau-eilbot = {
    enable = mkEnableOption "Tagesschau Breaking News bot for Telegram";

    package = mkPackageOption pkgs "tagesschau-eilbot" { };

    user = mkOption {
      type = types.str;
      default = defaultUser;
      description = "User under which RSS Bot runs.";
    };

    adminId = mkOption {
      type = types.int;
      description = "Admin ID";
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
        type = types.path;
        description = "Database user password file.";
      };
    };

  };

  config = mkIf cfg.enable {
    systemd.services.tagesschau-eilbot = {
      description = "RSS Bot for Telegram";
      after = [ "network.target" ];
      wantedBy = [ "multi-user.target" ];

      script = ''
        export BOT_TOKEN="$(< $CREDENTIALS_DIRECTORY/BOT_TOKEN )"
        export MYSQL_PASSWORD="$(< $CREDENTIALS_DIRECTORY/MYSQL_PASSWORD )"

        exec ${cfg.package}/bin/tagesschau-eilbot
      '';

      serviceConfig = {
        LoadCredential = [
          "BOT_TOKEN:${cfg.botTokenFile}"
          "MYSQL_PASSWORD:${cfg.database.passwordFile}"
        ];

        Restart = "always";
        User = cfg.user;
        Group = defaultUser;
      };

      environment = mkMerge [
        {
          ADMIN_ID = toString cfg.adminId;
          MYSQL_HOST = cfg.database.host;
          MYSQL_PORT = toString cfg.database.port;
          MYSQL_USER = cfg.database.user;
          MYSQL_DB = cfg.database.name;
        }
        (mkIf cfg.debug {
          DEBUG = "true";
        })
      ];
    };

    users = optionalAttrs (cfg.user == defaultUser) {
      users.${defaultUser} = {
        isSystemUser = true;
        group = defaultUser;
        description = "Tagesschau Breaking News Bot user";
      };

      groups.${defaultUser} = { };
    };

  };

}
