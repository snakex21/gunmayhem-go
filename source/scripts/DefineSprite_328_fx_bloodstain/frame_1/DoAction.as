gotoAndStop(random(_totalframes) + 1);
_rotation = random(360);
time = 0;
_xscale = 150;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _xscale = _xscale + 50;
      _yscale = _xscale;
      _alpha = _alpha - 20;
      if(_alpha <= 1 || time >= 4 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
