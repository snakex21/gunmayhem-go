stop();
Stage.showMenu = false;
total = _root.getBytesTotal();
this.onEnterFrame = function()
{
   loaded = _root.getBytesLoaded();
   percent = int(loaded / total * 100);
   if(loaded == total)
   {
      total = 0;
      play();
      delete this.onEnterFrame;
   }
};
