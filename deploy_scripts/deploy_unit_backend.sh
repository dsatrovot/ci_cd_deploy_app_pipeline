#! /usr/bin/env bash
set -Eeou pipefail
echo $#
if [[ $# -eq 0 ]]; then
        path_to_script="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

        service_files="$(find "$path_to_script" -type f -name "*.service")"
        IFS=$'\n'
        for file in $service_files; do
                env_file_name="$(basename -- "${file}" .service).env"
                full_path_to_dir="$(dirname -- "${file}")"
                if [[ ! -f "${full_path_to_dir}/${env_file_name}" ]]; then
                        touch "$full_path_to_dir/$env_file_name"
                        echo "kds"
                        chown pulseapp_user:pulseapp_user "$full_path_to_dir/$env_file_name"
                        chmod 440 "$full_path_to_dir/$env_file_name"
                fi
                chown pulseapp_user:pulseapp_user "$file"
                echo "DATABASE_URL=$DATABASE_URL" > "$full_path_to_dir/$env_file_name"
                chmod 644 "$file"
                cp -p "$file" "/home/pulseapp_user/.config/systemd/user/"
                cp -p "$full_path_to_dir/$env_file_name" "/home/pulseapp_user/.config/systemd/user/"
                rm "$file"
                rm "$full_path_to_dir/$env_file_name"
                su - pulseapp_user -c 'systemctl --user daemon-reload'
                echo "$(basename -- "${file}")" 
                su - pulseapp_user -c "systemctl --user enable "$(basename -- "${file}")"" 
                su - pulseapp_user -c "systemctl --user start "$(basename -- "${file}")""
        done
elif [[ $# -eq 1 ]]; then
        path_to_script="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
        env_file_name="$(basename -- $path_to_script/$1 .service).env"
        if [[ -f "$path_to_script/$1" ]]; then
                if [[ ! -f "$path_to_script/$env_file_name" ]]; then
                        touch "$path_to_script/$env_file_name"
                        chown pulseapp_user:pulseapp_user "$path_to_script/$env_file_name"
                        chmod 440 "$path_to_script/$env_file_name"
                fi
                echo "DATABASE_URL=$DATABASE_URL" > "$path_to_script/$env_file_name"

                chown pulseapp_user:pulseapp_user "$path_to_script/$1"

                chmod 644 "$path_to_script/$1"

                cp -p "$path_to_script/$1" "/home/pulseapp_user/.config/systemd/user/"
                cp -p "$path_to_script/$env_file_name" "/home/pulseapp_user/.config/systemd/user/"
                rm "$path_to_script/$1"
                rm "$path_to_script/$env_file_name"
                su - pulseapp_user -c 'systemctl --user daemon-reload'
                su - pulseapp_user -c "systemctl --user enable $1"
                su - pulseapp_user -c "systemctl --user start $1"
        else
                echo "Файла $1 не существует"
                exit 1
        fi
else
        echo "Ошибка"
        exit 1
fi