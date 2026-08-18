this.onEnterFrame = function()
{
   if(!_root.GAMEPAUSED)
   {
      _alpha = _alpha - 20;
      if(_alpha <= 1)
      {
         removeMovieClip(this);
         delete this.onEnterFrame;
      }
   }
};
