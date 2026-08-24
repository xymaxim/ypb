# Anticipated questions

Here are the answers to the most asked/anticipated questions.

## Why does the actual time differ from the target time?

The target time is the moment you requested, while the actual time is snapped to
the nearest matching segment boundary, start or end. The difference between the
two can be up to one full segment in length, with the segment duration depending
on the [YoutTube streaming
latency](https://support.google.com/youtube/answer/7444635?sjid=3264258360401641547-EU)
settings: 1 second (ultra-low latency), 2 seconds (low latency), or 5 seconds
for normal latency.

## Why is the output duration longer than requested?

Segments are merged as-is, without trimming the boundary segments to match the
requested interval. As a result, both boundaries can extend beyond the requested
interval (see [Why does the actual time differ from the target
time?](#why-does-the-actual-time-differ-from-the-target-time)). This means the
excerpt can be up to two full segments longer than requested: 2 seconds
(ultra-low latency), 4 seconds (low), or 10 seconds (normal).

## Why is the output duration shorter than requested?

Streams may contain gaps due to instability. A decrease in the duration of an
output excerpt (by seconds, minutes, or even hours) can result from: (a) the
start and/or end point falling within a gap, in which case the nearest available
segment is used instead, or (b) the excerpt containing one or more gaps
internally.
