#! /usr/bin/env bash
set -Eeou pipefail
shopt -s extglob
echo $#
IFS=$'\n'
if [[ $# -eq 0 ]]; then
        path_to_script="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

        nginx_files="$(find "$path_to_script" -type f -name "*.conf")"
        for file in $nginx_files; do

                chown root:root "$file"
                mv "$file" "/etc/nginx/sites-available/"
                file_name_without_dir="$(basename -- "$file")"
                if [[ ! -f "/etc/nginx/sites-enabled/$file_name_without_dir" ]]; then
                        ln -s "/etc/nginx/sites-available/$file_name_without_dir" "/etc/nginx/sites-enabled/$file_name_without_dir"
                fi


        done
        mv $path_to_script/!(deploy_unit.sh) /var/www/pulseapp/
        systemctl restart nginx.service
elif [[ $# -eq 1 ]]; then
        path_to_script="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
        if [[ -f "$path_to_script/$1" ]]; then


                chown root:root "$path_to_script/$1"



                mv "$path_to_script/$1" "/etc/nginx/sites-available/"
                if [[ ! -f "/etc/nginx/sites-enabled/$1" ]]; then

                        ln -s "/etc/nginx/sites-available/$1" "/etc/nginx/sites-enabled/$1"
                fi
                
                mv $path_to_script/!(deploy_unit.sh) /var/www/pulseapp/
                systemctl restart nginx.service
        else
                echo "Файла $1 не существует"
                exit 1
        fi
else
        echo "Ошибка"
        exit 1
fi