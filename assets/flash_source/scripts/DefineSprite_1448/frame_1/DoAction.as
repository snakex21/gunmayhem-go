fps = 40;
starttime = new Date();
lasttime = starttime.getMilliseconds();
counter = 0;
this.onEnterFrame = function()
{
   counter += 1;
   if(counter >= 10)
   {
      counter = 0;
      time = new Date();
      timepassed = time.getMilliseconds() - lasttime < 0 ? 1000 + (time.getMilliseconds() - lasttime) : time.getMilliseconds() - lasttime;
      fps = Math.round(10000 / timepassed);
      lasttime = time.getMilliseconds();
   }
};
