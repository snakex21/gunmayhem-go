stop();
this.onEnterFrame = function()
{
   if(_parent._x < 5)
   {
      delete this.onEnterFrame;
      play();
   }
};
