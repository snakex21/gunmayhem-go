_Y = _Y - 10;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _alpha = _alpha - 5;
      _xscale = _xscale - 5;
      _yscale = _xscale;
      _Y = _Y + 1;
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
