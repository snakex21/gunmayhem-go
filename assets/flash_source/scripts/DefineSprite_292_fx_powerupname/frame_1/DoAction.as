_alpha = 0;
level = 1;
time = 0;
gotoAndStop(asdf + 1);
this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _Y = _Y - 2;
      if(level == 1)
      {
         if(_alpha < 80)
         {
            _alpha = _alpha + 10;
         }
         time += 1;
         if(time >= 40)
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
