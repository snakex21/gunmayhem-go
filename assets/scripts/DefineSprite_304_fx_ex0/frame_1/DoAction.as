level = 1;
if(random(2) == 0)
{
   _rotation = _rotation * -1;
}
scaletarget = 95;
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      if(level == 1)
      {
         _xscale = _xscale + (100 - _xscale) / 3;
         _yscale = _xscale;
         if(_xscale >= scaletarget)
         {
            level = 2;
         }
      }
      if(level == 2)
      {
         _alpha = _alpha - 20;
      }
      if(_alpha <= 1 || _root.deleteeverything)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
