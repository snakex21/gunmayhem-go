level = 1;
_rotation = _rotation + (random(20) - 10);
scaletarget = 124;
_xscale = 10;
_yscale = xscale;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(level == 1)
      {
         _xscale = _xscale + (125 - _xscale) / 3;
         _yscale = _xscale;
         if(_xscale >= scaletarget)
         {
            level = 2;
         }
      }
      if(level == 2)
      {
         _alpha = _alpha - 10;
      }
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
