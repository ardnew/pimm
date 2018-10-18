#!/bin/bash
#
# various cleanup checks and stuff
#
clear

line="----"
printf "\n\n%s\n%s\n%s\n" "$line" "function definitions without a preceding comment:" "$line"
for i in *.go; do 
  grep '^func ' $i -B1|perl -ne'BEGIN{$P=0}$P=1if$.%3==1&&!m|^//|;$P=0if$.%3==0;print"'$i': ",$_ if$P==1&&$.%3==2;'
done
