level = 1;
scaletarget = 195;
_xscale = 10;
_yscale = _xscale;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(level == 1)
      {
         _xscale = _xscale + (200 - _xscale) / 3;
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
