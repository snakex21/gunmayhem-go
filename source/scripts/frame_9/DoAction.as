stop();
_X = 0;
_Y = 0;
_xscale = 100;
_yscale = 100;
gamewin = false;
crateON = true;
powerON = true;
this.onEnterFrame = function()
{
   _X = 0;
   _Y = 0;
   _xscale = 100;
   _yscale = 100;
   _quality = "HIGH";
   delete this.onEnterFrame;
};
