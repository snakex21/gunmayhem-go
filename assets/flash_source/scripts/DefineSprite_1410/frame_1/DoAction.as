this.onEnterFrame = function()
{
   _parent.swaptime += 1;
   if(_parent.swaptime >= _parent.swaptotal)
   {
      if(_parent.swaptotal >= 10)
      {
         _parent.randgun = dropgun._currentframe;
         delete this.onEnterFrame;
      }
      else
      {
         dropgun.gotoAndStop(random(57) + 10);
         _parent.randgun = dropgun._currentframe * -1;
      }
      _parent.swaptime = 0;
      _parent.swaptotal *= 1.25;
   }
};
