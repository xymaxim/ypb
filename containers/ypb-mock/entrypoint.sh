#!/bin/sh
set -e

VIDEO_RESOLUTIONS="134:640:360 135:854:480"

cp /usr/local/share/ypb-mock/info.json info.json

# Generate video segments
for res in $VIDEO_RESOLUTIONS; do
  itag=$(echo "$res" | cut -d: -f1)
  width=$(echo "$res" | cut -d: -f2)
  height=$(echo "$res" | cut -d: -f3)

  echo "Generating video itag=$itag (${width}x${height})..."
  ffmpeg -f lavfi -i "smptebars=duration=3600:size=${width}x${height}:rate=30" \
         -vf "drawtext=:text='%{pts\:hms}':x=(w-text_w)/2:y=(h-text_h)/2:fontcolor=white:fontsize=h/5:box=1:boxcolor=black" \
         -c:v libx264 -pix_fmt yuv420p \
         -video_track_timescale 90000 \
         -x264-params "keyint=150:min-keyint=150:scenecut=0" \
         -y "source-${itag}.mp4"

  mkdir -p "segments/${itag}"
  ffmpeg -i "source-${itag}.mp4" \
         -c copy -f segment -segment_time 5 -segment_start_number 1 \
         -segment_time_delta 0.05 \
         -reset_timestamps 1 \
         -movflags +frag_keyframe+empty_moov+default_base_moof \
         -segment_format_options movflags=+frag_keyframe+empty_moov+default_base_moof \
         -y "segments/${itag}/%d.mp4"

  rm "source-${itag}.mp4"
done

# Generate audio segments
echo "Generating audio itag=140..."
ffmpeg -f lavfi -i "aevalsrc=0.3*sin(2*PI*440*t)*lt(mod(t\,1)\,0.1):s=44100:d=3600" \
       -c:a aac -b:a 128k -ar 44100 \
       -y source-140.mp4

mkdir -p segments/140
ffmpeg -i source-140.mp4 \
       -c copy -f segment -segment_time 5 -segment_start_number 1 \
       -segment_time_delta 0.05 \
       -reset_timestamps 1 \
       -movflags +frag_keyframe+empty_moov+default_base_moof \
       -segment_format_options movflags=+frag_keyframe+empty_moov+default_base_moof \
       -y "segments/140/%d.mp4"

rm source-140.mp4
