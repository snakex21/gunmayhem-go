stop();
this.onEnterFrame = function()
{
   _alpha = _alpha - 10;
   if(_alpha <= 1)
   {
      _root.gotomenu = true;
      newmc = _root.attachMovie("fadeaway","fadeaway",_root.fadedepth);
      newmc.targetframe = 10;
      this.swapDepths(1);
      removeMovieClip(this);
      delete this.onEnterFrame;
   }
};
