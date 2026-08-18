stop();
this.onRollOver = function()
{
   if(_currentframe == 1)
   {
      gotoAndStop(2);
   }
};
this.onRollOut = function()
{
   if(_currentframe == 2)
   {
      gotoAndStop(1);
   }
};
this.useHandCursor = true;
